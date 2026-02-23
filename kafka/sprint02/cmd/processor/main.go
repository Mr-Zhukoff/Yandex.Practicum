package main

import (
	"context"
	"kafka-message-filter/processors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/lovoo/goka"
)

func getBrokers() []string {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv != "" {
		return strings.Split(brokersEnv, ",")
	}
	return []string{"localhost:9092"}
}

var (
	brokers = getBrokers()

	// Define Kafka topics/streams
	messagesStream       goka.Stream = "messages"
	filteredMessagesStream goka.Stream = "filtered_messages"
	blockActionsStream   goka.Stream = "blocked_users"
	censorActionsStream  goka.Stream = "censor-actions"
)

// ensureTopicsExist creates Kafka topics if they don't exist
func ensureTopicsExist() error {
	log.Println("Checking and creating Kafka topics if needed...")
	
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	
	// Retry connection to Kafka
	var admin sarama.ClusterAdmin
	var err error
	maxRetries := 10
	
	for i := 0; i < maxRetries; i++ {
		admin, err = sarama.NewClusterAdmin(brokers, config)
		if err == nil {
			break
		}
		log.Printf("Waiting for Kafka to be ready... (attempt %d/%d)", i+1, maxRetries)
		time.Sleep(5 * time.Second)
	}
	
	if err != nil {
		return err
	}
	defer admin.Close()
	
	// Get existing topics
	existingTopics, err := admin.ListTopics()
	if err != nil {
		return err
	}
	
	// Define topics to create
	topicsToCreate := map[string]sarama.TopicDetail{
		"messages": {
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
		"filtered_messages": {
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
		"blocked_users": {
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
		"censor-actions": {
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}
	
	// Create topics that don't exist
	for topicName, topicDetail := range topicsToCreate {
		if _, exists := existingTopics[topicName]; !exists {
			log.Printf("Creating topic: %s", topicName)
			err := admin.CreateTopic(topicName, &topicDetail, false)
			if err != nil {
				log.Printf("Warning: Could not create topic %s: %v", topicName, err)
			} else {
				log.Printf("Topic %s created successfully", topicName)
			}
		} else {
			log.Printf("Topic %s already exists", topicName)
		}
	}
	
	log.Println("Topic check/creation complete")
	
	// Wait for topics to be fully propagated in the cluster
	log.Println("Waiting for topics to be fully ready...")
	time.Sleep(10 * time.Second)
	
	return nil
}

func main() {
	log.Println("Starting Kafka Message Filter System...")
	
	// Ensure topics exist before starting processors
	if err := ensureTopicsExist(); err != nil {
		log.Fatalf("Failed to ensure topics exist: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and start block manager and censor manager first
	// They need to create their group tables before message filter can use them
	log.Println("Creating block manager processor...")
	blockManagerProc := processors.DefineBlockManagerGroup(brokers, blockActionsStream)

	log.Println("Creating censor manager processor...")
	censorManagerProc := processors.DefineCensorManagerGroup(brokers, censorActionsStream)

	// Start block manager and censor manager
	log.Println("Starting block manager and censor manager...")
	
	errChan := make(chan error, 3)

	// Start block manager
	go func() {
		log.Println("Block manager processor starting...")
		if err := blockManagerProc.Run(ctx); err != nil {
			errChan <- err
		}
	}()

	// Start censor manager
	go func() {
		log.Println("Censor manager processor starting...")
		if err := censorManagerProc.Run(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for group tables to be created
	log.Println("Waiting for group tables to be created...")
	time.Sleep(15 * time.Second)

	// Now create and start message filter
	log.Println("Creating message filter processor...")
	messageFilterProc := processors.DefineMessageFilterGroup(
		brokers,
		messagesStream,
		filteredMessagesStream,
	)

	// Start message filter
	go func() {
		log.Println("Message filter processor starting...")
		if err := messageFilterProc.Run(ctx); err != nil {
			errChan <- err
		}
	}()

	log.Println("All processors started successfully!")
	log.Println("System is ready to process messages...")
	log.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigterm:
		log.Println("Shutdown signal received, stopping processors...")
		cancel()
	case err := <-errChan:
		log.Printf("Error occurred: %v", err)
		cancel()
	}

	log.Println("Waiting for processors to shut down...")
	// Give processors time to shut down gracefully
	<-ctx.Done()
	log.Println("System stopped")
}
