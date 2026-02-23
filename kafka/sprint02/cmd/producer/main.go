package main

import (
	"encoding/json"
	"fmt"
	"kafka-message-filter/models"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
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
)

func main() {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	fmt.Println("=== Kafka Message Filter Test Producer ===")
	fmt.Println("This will send test data to demonstrate the system functionality")
	fmt.Println()

	// Wait a bit for system to be ready
	time.Sleep(2 * time.Second)

	// Test 1: Add censored words
	fmt.Println("Step 1: Adding censored words...")
	sendCensorAction(producer, "badword", "add")
	sendCensorAction(producer, "spam", "add")
	sendCensorAction(producer, "offensive", "add")
	time.Sleep(2 * time.Second)

	// Test 2: Send normal messages
	fmt.Println("\nStep 2: Sending normal messages (should pass through)...")
	sendMessage(producer, "msg1", "alice", "bob", "Hello Bob, how are you?")
	sendMessage(producer, "msg2", "bob", "alice", "Hi Alice, I'm doing great!")
	time.Sleep(2 * time.Second)

	// Test 3: Send messages with censored words
	fmt.Println("\nStep 3: Sending messages with censored words (should be censored)...")
	sendMessage(producer, "msg3", "charlie", "bob", "This is a badword message!")
	sendMessage(producer, "msg4", "alice", "bob", "No spam here, just testing")
	sendMessage(producer, "msg5", "dave", "alice", "This contains offensive content")
	time.Sleep(2 * time.Second)

	// Test 4: Block a user
	fmt.Println("\nStep 4: Bob blocks Charlie...")
	sendBlockAction(producer, "bob", "charlie", "block")
	time.Sleep(2 * time.Second)

	// Test 5: Send message from blocked user
	fmt.Println("\nStep 5: Sending message from blocked user (should be blocked)...")
	sendMessage(producer, "msg6", "charlie", "bob", "Hello Bob, can you see this?")
	time.Sleep(1 * time.Second)

	// Test 6: Send message from non-blocked user
	fmt.Println("\nStep 6: Sending message from non-blocked user (should pass)...")
	sendMessage(producer, "msg7", "alice", "bob", "Hello Bob from Alice")
	time.Sleep(2 * time.Second)

	// Test 7: Alice blocks Dave
	fmt.Println("\nStep 7: Alice blocks Dave...")
	sendBlockAction(producer, "alice", "dave", "block")
	time.Sleep(2 * time.Second)

	// Test 8: Send message from Dave to Alice (should be blocked)
	fmt.Println("\nStep 8: Dave tries to message Alice (should be blocked)...")
	sendMessage(producer, "msg8", "dave", "alice", "Hi Alice!")
	time.Sleep(1 * time.Second)

	// Test 9: Unblock user
	fmt.Println("\nStep 9: Bob unblocks Charlie...")
	sendBlockAction(producer, "bob", "charlie", "unblock")
	time.Sleep(2 * time.Second)

	// Test 10: Send message from previously blocked user
	fmt.Println("\nStep 10: Charlie messages Bob again (should now pass)...")
	sendMessage(producer, "msg9", "charlie", "bob", "Thanks for unblocking me Bob!")
	time.Sleep(1 * time.Second)

	// Test 11: Message with censored word from previously blocked user
	fmt.Println("\nStep 11: Charlie sends message with censored word (should be censored but delivered)...")
	sendMessage(producer, "msg10", "charlie", "bob", "Sorry for the badword earlier")
	time.Sleep(1 * time.Second)

	// Test 12: Remove censored word
	fmt.Println("\nStep 12: Removing 'spam' from censored words...")
	sendCensorAction(producer, "spam", "remove")
	time.Sleep(2 * time.Second)

	// Test 13: Send message with previously censored word
	fmt.Println("\nStep 13: Sending message with 'spam' (should not be censored now)...")
	sendMessage(producer, "msg11", "alice", "bob", "This spam word should be visible now")
	time.Sleep(1 * time.Second)

	fmt.Println("\n=== All test messages sent! ===")
	fmt.Println("Check the processor logs to see the filtering in action")
	fmt.Println("Check Kafka UI at http://localhost:8080 to see the messages in topics")
}

func sendMessage(producer sarama.SyncProducer, id, from, to, content string) {
	msg := &models.Message{
		ID:        id,
		From:      from,
		To:        to,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	// Use 'from' as key for partitioning
	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: "messages",
		Key:   sarama.StringEncoder(from),
		Value: sarama.ByteEncoder(data),
	})

	if err != nil {
		log.Printf("Failed to send message: %v", err)
	} else {
		fmt.Printf("  ✓ Sent message %s: %s -> %s: '%s'\n", id, from, to, content)
	}
}

func sendBlockAction(producer sarama.SyncProducer, blockerID, blockedID, action string) {
	blockAction := &models.BlockAction{
		BlockerID: blockerID,
		BlockedID: blockedID,
		Action:    action,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(blockAction)
	if err != nil {
		log.Printf("Failed to marshal block action: %v", err)
		return
	}

	// Use blockerID as key so all block actions for a user go to same partition
	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: "blocked_users",
		Key:   sarama.StringEncoder(blockerID),
		Value: sarama.ByteEncoder(data),
	})

	if err != nil {
		log.Printf("Failed to send block action: %v", err)
	} else {
		fmt.Printf("  ✓ Sent block action: %s %ss %s\n", blockerID, action, blockedID)
	}
}

func sendCensorAction(producer sarama.SyncProducer, word, action string) {
	censorAction := &models.CensorAction{
		Word:      word,
		Action:    action,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(censorAction)
	if err != nil {
		log.Printf("Failed to marshal censor action: %v", err)
		return
	}

	// Use "global" as key since censored words list is global
	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: "censor-actions",
		Key:   sarama.StringEncoder("global"),
		Value: sarama.ByteEncoder(data),
	})

	if err != nil {
		log.Printf("Failed to send censor action: %v", err)
	} else {
		fmt.Printf("  ✓ Sent censor action: %s word '%s'\n", action, word)
	}
}
