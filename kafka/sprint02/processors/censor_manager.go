package processors

import (
	"fmt"
	"kafka-message-filter/models"
	"log"
	"strings"

	"github.com/lovoo/goka"
)

const (
	CensoredWordsTable goka.Table = "censored-words-table"
	CensoredWordsKey   string     = "global" // Single key for global censored words list
)

// ProcessCensorAction processes add/remove censored word actions
func ProcessCensorAction(ctx goka.Context, msg interface{}) {
	action, ok := msg.(*models.CensorAction)
	if !ok {
		log.Printf("Error: invalid message type in censor action processor")
		return
	}

	log.Printf("Processing censor action: word=%s, action=%s", action.Word, action.Action)

	// Get current censored words list
	val := ctx.Value()
	var censoredList *models.CensoredWordsList
	
	if val == nil {
		censoredList = &models.CensoredWordsList{
			Words: make(map[string]models.CensoredWord),
		}
	} else {
		censoredList = val.(*models.CensoredWordsList)
		if censoredList.Words == nil {
			censoredList.Words = make(map[string]models.CensoredWord)
		}
	}

	// Update the list based on action
	wordLower := strings.ToLower(action.Word)
	switch action.Action {
	case "add":
		censoredList.Words[wordLower] = models.CensoredWord{
			Word:      action.Word,
			Timestamp: action.Timestamp,
		}
		log.Printf("Added censored word: %s", action.Word)
	case "remove":
		delete(censoredList.Words, wordLower)
		log.Printf("Removed censored word: %s", action.Word)
	default:
		log.Printf("Unknown action: %s", action.Action)
		return
	}

	// Store updated list back to the table
	ctx.SetValue(censoredList)
	log.Printf("Updated censored words list: %d words", len(censoredList.Words))
}

// CensorMessage applies censorship to message content
func CensorMessage(ctx goka.Context, content string) string {
	// Lookup the censored words list
	val := ctx.Lookup(CensoredWordsTable, CensoredWordsKey)

	if val == nil {
		return content
	}

	censoredList, ok := val.(*models.CensoredWordsList)
	if !ok {
		log.Printf("Error: invalid censored list type")
		return content
	}

	if len(censoredList.Words) == 0 {
		return content
	}

	// Apply censorship by replacing words with asterisks
	words := strings.Fields(content)
	censored := false
	
	for i, word := range words {
		// Remove punctuation for comparison
		cleanWord := strings.ToLower(strings.Trim(word, ".,!?;:\"'"))
		if _, isCensored := censoredList.Words[cleanWord]; isCensored {
			// Replace with asterisks of same length
			words[i] = strings.Repeat("*", len(word))
			censored = true
		}
	}

	if censored {
		result := strings.Join(words, " ")
		log.Printf("Censored message: '%s' -> '%s'", content, result)
		return result
	}

	return content
}

// GetCensoredWordsCount returns the count of censored words
func GetCensoredWordsCount(ctx goka.Context) int {
	val := ctx.Lookup(CensoredWordsTable, CensoredWordsKey)
	if val == nil {
		return 0
	}
	censoredList, ok := val.(*models.CensoredWordsList)
	if !ok {
		return 0
	}
	return len(censoredList.Words)
}

// DefineCensorManagerGroup defines the Goka group for managing censored words
func DefineCensorManagerGroup(brokers []string, censorActionsStream goka.Stream) *goka.Processor {
	group := goka.Group("censor-manager")
	
	g := goka.DefineGroup(
		group,
		goka.Input(censorActionsStream, new(models.CensorActionCodec), ProcessCensorAction),
		goka.Persist(new(models.CensoredWordsListCodec)),
	)

	// Configure for single broker (replication factor = 1)
	tmc := goka.NewTopicManagerConfig()
	tmc.Table.Replication = 1
	
	proc, err := goka.NewProcessor(brokers, g, goka.WithTopicManagerBuilder(
		goka.TopicManagerBuilderWithTopicManagerConfig(tmc),
	))
	if err != nil {
		log.Fatalf("Error creating censor manager processor: %v", err)
	}

	return proc
}

// PrintCensoredWords is a helper to print censored words (for debugging)
func PrintCensoredWords(censoredList *models.CensoredWordsList) {
	if censoredList == nil || len(censoredList.Words) == 0 {
		fmt.Println("No censored words")
		return
	}
	fmt.Printf("Censored words (%d):\n", len(censoredList.Words))
	for _, word := range censoredList.Words {
		fmt.Printf("  - %s (added at: %d)\n", word.Word, word.Timestamp)
	}
}
