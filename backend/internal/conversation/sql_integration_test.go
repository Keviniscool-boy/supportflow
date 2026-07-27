package conversation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	appclock "github.com/Keviniscool-boy/supportflow/backend/internal/clock"
	"github.com/Keviniscool-boy/supportflow/backend/internal/testfixture"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestSQLRepositoryMessageSequenceAndOwnership(t *testing.T) {
	databaseURL := os.Getenv("SUPPORTFLOW_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SUPPORTFLOW_TEST_DATABASE_URL is not set")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewSQLRepository(database), appclock.NewFixed(testfixture.FrozenTime))
	customer := fixtureCustomer()
	created, err := service.Create(ctx, customer, "SQL integration test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = database.ExecContext(cleanupContext, "DELETE FROM messages WHERE tenant_id = $1 AND conversation_id = $2", customer.TenantID, created.ID)
		_, _ = database.ExecContext(cleanupContext, "DELETE FROM conversations WHERE tenant_id = $1 AND id = $2", customer.TenantID, created.ID)
	})

	const messageCount = 10
	errorsChannel := make(chan error, messageCount)
	var waitGroup sync.WaitGroup
	for index := 0; index < messageCount; index++ {
		waitGroup.Add(1)
		go func(messageIndex int) {
			defer waitGroup.Done()
			_, appendErr := service.AppendMessage(ctx, customer, created.ID, NewMessage{
				ActorType: ActorCustomer, ContentType: ContentText, Text: fmt.Sprintf("message %d", messageIndex), Locale: "en-US",
			})
			errorsChannel <- appendErr
		}(index)
	}
	waitGroup.Wait()
	close(errorsChannel)
	for appendErr := range errorsChannel {
		if appendErr != nil {
			t.Fatal(appendErr)
		}
	}

	page, err := service.ListMessages(ctx, customer, created.ID, 0, 20)
	if err != nil || len(page.Items) != messageCount || page.HasMore {
		t.Fatalf("unexpected SQL message page: count=%d more=%v err=%v", len(page.Items), page.HasMore, err)
	}
	sequences := make([]int, 0, len(page.Items))
	for _, message := range page.Items {
		sequences = append(sequences, int(message.SequenceNo))
	}
	sort.Ints(sequences)
	for index, sequence := range sequences {
		if sequence != index+1 {
			t.Fatalf("message sequence is not contiguous: %v", sequences)
		}
	}

	otherCustomer := customer
	otherCustomer.CustomerID = "00000000-0000-0000-0000-000000000099"
	if _, err := service.Get(ctx, otherCustomer, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected SQL ownership rejection, got %v", err)
	}
}
