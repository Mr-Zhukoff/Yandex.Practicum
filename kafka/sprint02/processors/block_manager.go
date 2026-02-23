package processors

import (
	"fmt"
	"kafka-message-filter/models"
	"log"

	"github.com/lovoo/goka"
)

const (
	BlockedUsersTable goka.Table = "blocked-users-table"
)

// ProcessBlockAction processes block/unblock actions and stores them in group table
func ProcessBlockAction(ctx goka.Context, msg interface{}) {
	action, ok := msg.(*models.BlockAction)
	if !ok {
		log.Printf("Error: invalid message type in block action processor")
		return
	}

	log.Printf("Processing block action: blocker=%s, blocked=%s, action=%s",
		action.BlockerID, action.BlockedID, action.Action)

	// Get current blocked users list for the blocker
	val := ctx.Value()
	var blockedList *models.BlockedUsersList
	
	if val == nil {
		blockedList = &models.BlockedUsersList{
			Users: make(map[string]models.BlockedUser),
		}
	} else {
		blockedList = val.(*models.BlockedUsersList)
		if blockedList.Users == nil {
			blockedList.Users = make(map[string]models.BlockedUser)
		}
	}

	// Update the list based on action
	switch action.Action {
	case "block":
		blockedList.Users[action.BlockedID] = models.BlockedUser{
			UserID:    action.BlockedID,
			Timestamp: action.Timestamp,
		}
		log.Printf("User %s blocked user %s", action.BlockerID, action.BlockedID)
	case "unblock":
		delete(blockedList.Users, action.BlockedID)
		log.Printf("User %s unblocked user %s", action.BlockerID, action.BlockedID)
	default:
		log.Printf("Unknown action: %s", action.Action)
		return
	}

	// Store updated list back to the table
	ctx.SetValue(blockedList)
	log.Printf("Updated blocked list for user %s: %d blocked users",
		action.BlockerID, len(blockedList.Users))
}

// IsUserBlocked checks if senderID is blocked by receiverID
func IsUserBlocked(ctx goka.Context, receiverID, senderID string) bool {
	// Lookup the blocked users list for the receiver
	val := ctx.Lookup(BlockedUsersTable, receiverID)

	if val == nil {
		return false
	}

	blockedList, ok := val.(*models.BlockedUsersList)
	if !ok {
		log.Printf("Error: invalid blocked list type")
		return false
	}

	_, blocked := blockedList.Users[senderID]
	if blocked {
		log.Printf("User %s is blocked by %s", senderID, receiverID)
	}
	return blocked
}

// GetBlockedUsersCount returns the count of blocked users for a user
func GetBlockedUsersCount(ctx goka.Context, userID string) int {
	val := ctx.Lookup(BlockedUsersTable, userID)
	if val == nil {
		return 0
	}
	blockedList, ok := val.(*models.BlockedUsersList)
	if !ok {
		return 0
	}
	return len(blockedList.Users)
}

// DefineBlockManagerGroup defines the Goka group for managing blocked users
func DefineBlockManagerGroup(brokers []string, blockActionsStream goka.Stream) *goka.Processor {
	group := goka.Group("block-manager")
	
	g := goka.DefineGroup(
		group,
		goka.Input(blockActionsStream, new(models.BlockActionCodec), ProcessBlockAction),
		goka.Persist(new(models.BlockedUsersListCodec)),
	)

	// Configure for single broker (replication factor = 1)
	tmc := goka.NewTopicManagerConfig()
	tmc.Table.Replication = 1
	
	proc, err := goka.NewProcessor(brokers, g, goka.WithTopicManagerBuilder(
		goka.TopicManagerBuilderWithTopicManagerConfig(tmc),
	))
	if err != nil {
		log.Fatalf("Error creating block manager processor: %v", err)
	}

	return proc
}

// PrintBlockedUsers is a helper to print blocked users (for debugging)
func PrintBlockedUsers(blockedList *models.BlockedUsersList) {
	if blockedList == nil || len(blockedList.Users) == 0 {
		fmt.Println("No blocked users")
		return
	}
	fmt.Printf("Blocked users (%d):\n", len(blockedList.Users))
	for userID, user := range blockedList.Users {
		fmt.Printf("  - %s (blocked at: %d)\n", userID, user.Timestamp)
	}
}
