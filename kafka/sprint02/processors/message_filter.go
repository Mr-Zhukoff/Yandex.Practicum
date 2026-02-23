package processors

import (
	"kafka-message-filter/models"
	"log"

	"github.com/lovoo/goka"
)

// ProcessMessage filters and censors incoming messages
func ProcessMessage(ctx goka.Context, msg interface{}) {
	message, ok := msg.(*models.Message)
	if !ok {
		log.Printf("Error: invalid message type in message processor")
		return
	}

	log.Printf("Processing message: id=%s, from=%s, to=%s, content=%s",
		message.ID, message.From, message.To, message.Content)

	// Check if sender is blocked by receiver
	if IsUserBlocked(ctx, message.To, message.From) {
		log.Printf("Message %s blocked: sender %s is blocked by receiver %s",
			message.ID, message.From, message.To)
		return
	}

	// Apply censorship to message content
	originalContent := message.Content
	message.Content = CensorMessage(ctx, message.Content)
	
	if originalContent != message.Content {
		log.Printf("Message %s censored: '%s' -> '%s'",
			message.ID, originalContent, message.Content)
	}

	// Emit filtered message to output stream
	ctx.Emit(goka.Stream("filtered_messages"), message.To, message)
	log.Printf("Message %s forwarded to receiver %s", message.ID, message.To)
}

// DefineMessageFilterGroup defines the Goka group for filtering messages
func DefineMessageFilterGroup(
	brokers []string,
	messagesStream goka.Stream,
	filteredMessagesStream goka.Stream,
) *goka.Processor {
	group := goka.Group("message-filter")
	
	g := goka.DefineGroup(
		group,
		goka.Input(messagesStream, new(models.MessageCodec), ProcessMessage),
		goka.Output(filteredMessagesStream, new(models.MessageCodec)),
		goka.Lookup(BlockedUsersTable, new(models.BlockedUsersListCodec)),
		goka.Lookup(CensoredWordsTable, new(models.CensoredWordsListCodec)),
	)

	// Configure for single broker (replication factor = 1)
	tmc := goka.NewTopicManagerConfig()
	tmc.Table.Replication = 1
	
	proc, err := goka.NewProcessor(brokers, g, goka.WithTopicManagerBuilder(
		goka.TopicManagerBuilderWithTopicManagerConfig(tmc),
	))
	if err != nil {
		log.Fatalf("Error creating message filter processor: %v", err)
	}

	return proc
}
