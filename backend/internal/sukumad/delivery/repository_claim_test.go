package delivery

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMemoryRepositoryClaimNextPendingDeliveryOnlyClaimsOnce(t *testing.T) {
	repo := NewRepository()
	created, err := repo.CreateDelivery(context.Background(), CreateParams{
		UID:           "delivery-1",
		RequestID:     11,
		ServerID:      7,
		AttemptNumber: 1,
		Status:        StatusPending,
	})
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	var (
		wg      sync.WaitGroup
		results = make(chan Record, 2)
		errs    = make(chan error, 2)
	)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			record, claimErr := repo.ClaimNextPendingDelivery(context.Background(), time.Now().UTC())
			if claimErr != nil {
				errs <- claimErr
				return
			}
			results <- record
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	var claimed []Record
	for record := range results {
		claimed = append(claimed, record)
	}
	var noEligible int
	for err := range errs {
		if errors.Is(err, ErrNoEligibleDelivery) {
			noEligible++
			continue
		}
		t.Fatalf("unexpected claim error: %v", err)
	}
	if len(claimed) != 1 || noEligible != 1 {
		t.Fatalf("expected one claim and one no-eligible result, got claimed=%d noEligible=%d", len(claimed), noEligible)
	}
	if claimed[0].ID != created.ID || claimed[0].Status != StatusRunning {
		t.Fatalf("unexpected claimed record: %+v", claimed[0])
	}
}

func TestMemoryRepositoryClaimNextRetryDeliveryHonorsRetryAt(t *testing.T) {
	repo := NewRepository()
	future := time.Now().UTC().Add(2 * time.Minute)
	if _, err := repo.CreateDelivery(context.Background(), CreateParams{
		UID:           "retry-later",
		RequestID:     9,
		ServerID:      3,
		AttemptNumber: 2,
		Status:        StatusRetrying,
		RetryAt:       &future,
	}); err != nil {
		t.Fatalf("create future retry delivery: %v", err)
	}
	due := time.Now().UTC().Add(-time.Minute)
	created, err := repo.CreateDelivery(context.Background(), CreateParams{
		UID:           "retry-now",
		RequestID:     9,
		ServerID:      3,
		AttemptNumber: 3,
		Status:        StatusRetrying,
		RetryAt:       &due,
	})
	if err != nil {
		t.Fatalf("create due retry delivery: %v", err)
	}

	record, err := repo.ClaimNextRetryDelivery(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("claim retry delivery: %v", err)
	}
	if record.ID != created.ID || record.Status != StatusRunning || record.RetryAt != nil {
		t.Fatalf("unexpected claimed retry record: %+v", record)
	}
}
