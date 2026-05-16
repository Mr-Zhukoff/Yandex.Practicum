package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lovoo/goka"

	"marketplace-analytics/internal/events"
	"marketplace-analytics/internal/gokautil"
	"marketplace-analytics/internal/kafkautil"
)

func main() {
	brokersCSV := flag.String("brokers", "localhost:9092", "comma-separated Kafka brokers")
	rawTopic := flag.String("raw-topic", "shop.products.raw", "raw products topic")
	allowedTopic := flag.String("allowed-topic", "shop.products.allowed", "allowed products topic")
	rejectedTopic := flag.String("rejected-topic", "shop.products.rejected", "rejected products topic")
	dlqTopic := flag.String("dlq-topic", "shop.products.dlq", "dead-letter topic")
	forbiddenTopic := flag.String("forbidden-topic", "forbidden.products.state", "forbidden products compacted table topic")
	groupID := flag.String("group", "product-filter", "Goka processor group")
	flag.Parse()

	brokers := kafkautil.Brokers(*brokersCSV)
	group := goka.Group(*groupID)
	rawStream := goka.Stream(*rawTopic)
	allowedStream := goka.Stream(*allowedTopic)
	rejectedStream := goka.Stream(*rejectedTopic)
	dlqStream := goka.Stream(*dlqTopic)
	forbiddenTable := goka.Table(*forbiddenTopic)

	productCodec := gokautil.JSONCodec[events.EventEnvelope[events.Product]]{}
	rejectedCodec := gokautil.JSONCodec[events.EventEnvelope[events.RejectedProduct]]{}
	dlqCodec := gokautil.JSONCodec[events.EventEnvelope[events.DeadLetter]]{}
	forbiddenCodec := gokautil.JSONCodec[events.ForbiddenProduct]{}

	graph := goka.DefineGroup(group,
		goka.Input(rawStream, productCodec, func(ctx goka.Context, msg interface{}) {
			processProduct(ctx, msg, forbiddenTable, allowedStream, rejectedStream, dlqStream)
		}),
		goka.Output(allowedStream, productCodec),
		goka.Output(rejectedStream, rejectedCodec),
		goka.Output(dlqStream, dlqCodec),
		goka.Lookup(forbiddenTable, forbiddenCodec),
	)

	processor, err := goka.NewProcessor(brokers, graph)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("Goka product-filter started: input=%s allowed=%s rejected=%s dlq=%s forbidden-table=%s", *rawTopic, *allowedTopic, *rejectedTopic, *dlqTopic, *forbiddenTopic)
	if err := processor.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}

func processProduct(ctx goka.Context, msg interface{}, forbiddenTable goka.Table, allowedStream, rejectedStream, dlqStream goka.Stream) {
	envelope, ok := msg.(*events.EventEnvelope[events.Product])
	if !ok || envelope == nil {
		emitDLQ(ctx, dlqStream, "", "unexpected product message type")
		return
	}

	product := envelope.Payload
	if err := events.ValidateProduct(product); err != nil {
		emitDLQ(ctx, dlqStream, ctx.Key(), "invalid product: "+err.Error())
		return
	}

	if isForbidden(ctx, forbiddenTable, product.ProductID) {
		rejected := events.EventEnvelope[events.RejectedProduct]{
			EventID:   envelope.EventID,
			EventType: "product_rejected",
			EventTime: time.Now().UTC(),
			Source:    "product-filter",
			Payload: events.RejectedProduct{
				Product: product,
				Reason:  "product is forbidden",
			},
		}
		ctx.Emit(rejectedStream, product.ProductID, rejected)
		log.Printf("rejected forbidden product %s", product.ProductID)
		return
	}

	ctx.Emit(allowedStream, product.ProductID, envelope)
	log.Printf("allowed product %s", product.ProductID)
}

func isForbidden(ctx goka.Context, table goka.Table, productID string) bool {
	value := ctx.Lookup(table, productID)
	if value == nil {
		return false
	}
	record, ok := value.(*events.ForbiddenProduct)
	return ok && record.Active
}

func emitDLQ(ctx goka.Context, stream goka.Stream, key, reason string) {
	if key == "" {
		key = ctx.Key()
	}
	dlq := events.EventEnvelope[events.DeadLetter]{
		EventID:   key,
		EventType: "dead_letter",
		EventTime: time.Now().UTC(),
		Source:    "product-filter",
		Payload: events.DeadLetter{
			RawPayload: "",
			Reason:     reason,
			Source:     "shop.products.raw",
		},
	}
	ctx.Emit(stream, key, dlq)
	log.Printf("sent product event to DLQ: key=%s reason=%s", key, reason)
}
