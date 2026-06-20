package config

import (
	"context"
	"testing"
	"time"
)

func TestMongoBackoff(t *testing.T) {
	// Attempt 1: backoff should be around 200ms
	b := mongoBackoff(1)
	if b < 200*time.Millisecond || b > 300*time.Millisecond {
		t.Errorf("expected backoff ~200ms, got %v", b)
	}

	// Attempt 0: defaults to 1
	b0 := mongoBackoff(0)
	if b0 < 200*time.Millisecond || b0 > 300*time.Millisecond {
		t.Errorf("expected backoff ~200ms for attempt 0, got %v", b0)
	}

	// Attempt 10: should hit max backoff (5s)
	bMax := mongoBackoff(10)
	if bMax < 5*time.Second || bMax > 6*time.Second { // considering jitter
		t.Errorf("expected backoff ~5s, got %v", bMax)
	}
}

func TestCryptoJitter(t *testing.T) {
	j, err := cryptoJitter(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j < 0 || j >= 100*time.Millisecond {
		t.Errorf("expected jitter between 0 and 100ms, got %v", j)
	}

	// zero or negative max
	j0, err := cryptoJitter(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if j0 != 0 {
		t.Errorf("expected 0 jitter, got %v", j0)
	}
}

func TestAttemptMongoReconnect_NoConfig(t *testing.T) {
	// Clear the config
	lastMongoCfg.Store(mongoCfg{})

	// This should not panic and return early
	attemptMongoReconnect()
}

func TestConnectMongoWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := mongoCfg{uri: "mongodb://invalid", dbName: "db", collName: "coll"}

	_, _, err := connectMongoWithRetry(ctx, cfg, 5*time.Second)
	if err != context.Canceled {
		t.Errorf("expected context canceled, got %v", err)
	}
}

func TestConnectMongoOnce_InvalidURI(t *testing.T) {
	cfg := mongoCfg{uri: "http://invalid-mongo-uri", dbName: "db", collName: "coll"}
	_, _, err := connectMongoOnce(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for invalid URI")
	}
}
