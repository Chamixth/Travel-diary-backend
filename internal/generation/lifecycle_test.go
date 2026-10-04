package generation

import (
	"context"
	"testing"
	"time"
)

func TestQueueShutdownIsIdempotentWithoutActiveWorkers(t *testing.T) {
	t.Parallel()
	claimContext, stopClaims := context.WithCancel(context.Background())
	executionContext, cancelExecutions := context.WithCancel(context.Background())
	queue := &Queue{
		wake: make(chan struct{}, 1), claimContext: claimContext, stopClaims: stopClaims,
		executionContext: executionContext, cancelExecutions: cancelExecutions,
		shutdownDone: make(chan struct{}),
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := queue.Shutdown(shutdownContext); err != nil {
		t.Fatal(err)
	}
	if err := queue.Shutdown(shutdownContext); err != nil {
		t.Fatalf("second shutdown must be safe: %v", err)
	}
	if claimContext.Err() == nil || executionContext.Err() == nil {
		t.Fatal("shutdown must cancel claim and execution contexts")
	}
}

func TestNormalizedMaxAttemptsUsesSafeBounds(t *testing.T) {
	t.Parallel()
	if got := normalizedMaxAttempts(5); got != 5 {
		t.Fatalf("expected configured value, got %d", got)
	}
	for _, invalid := range []int{-1, 0, 11} {
		if got := normalizedMaxAttempts(invalid); got != defaultMaxAttempts {
			t.Fatalf("expected default for %d, got %d", invalid, got)
		}
	}
}
