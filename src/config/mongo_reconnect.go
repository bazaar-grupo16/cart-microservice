package config

import (
	"context"
	crand "crypto/rand"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
	"go.uber.org/zap"
)

const (
	mongoRetryTimeout   = 30 * time.Second
	mongoPingTimeout    = 3 * time.Second
	mongoIndexTimeout   = 5 * time.Second
	mongoInitialBackoff = 200 * time.Millisecond
	mongoMaxBackoff     = 5 * time.Second
	mongoMonitorPeriod  = 5 * time.Second
)

var (
	monitorOnce  sync.Once
	reconnecting atomic.Bool
)

func startMongoReconnectMonitor() {
	monitorOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(mongoMonitorPeriod)
			defer ticker.Stop()

			for range ticker.C {
				client := GetMongoClient()
				if client == nil {
					continue
				}

				pingCtx, cancel := context.WithTimeout(context.Background(), mongoPingTimeout)
				err := client.Ping(pingCtx, nil)
				cancel()
				if err == nil {
					continue
				}

				zap.S().Warnw("MongoDB ping failed; attempting reconnect", "error", err)
				attemptMongoReconnect()
			}
		}()
	})
}

func attemptMongoReconnect() {
	if reconnecting.Swap(true) {
		return
	}
	defer reconnecting.Store(false)

	cfgAny := lastMongoCfg.Load()
	if cfgAny == nil {
		zap.S().Warn("MongoDB reconnect skipped: no config available")
		return
	}
	cfg := cfgAny.(mongoCfg)

	newClient, newCollection, err := connectMongoWithRetry(context.Background(), cfg, mongoRetryTimeout)
	if err != nil {
		zap.S().Errorw("MongoDB reconnection failed after retries", "error", err)
		return
	}

	oldClient := GetMongoClient()
	setMongoGlobals(newClient, newCollection)

	if oldClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := oldClient.Disconnect(ctx); err != nil {
			zap.S().Warnw("Failed to disconnect old MongoDB client", "error", err)
		}
		cancel()
	}

	zap.S().Infow("MongoDB reconnected", "db", cfg.dbName, "collection", cfg.collName)
}

func connectMongoWithRetry(parent context.Context, cfg mongoCfg, timeout time.Duration) (*mongo.Client, *mongo.Collection, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		attempt++
		client, collection, err := connectMongoOnce(ctx, cfg)
		if err == nil {
			return client, collection, nil
		}

		backoff := mongoBackoff(attempt)
		zap.S().Warnw("MongoDB connect failed; retrying",
			"attempt", attempt,
			"backoff", backoff.String(),
			"error", err,
		)

		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, nil, ctx.Err()
		case <-t.C:
		}
	}
}

func connectMongoOnce(parent context.Context, cfg mongoCfg) (*mongo.Client, *mongo.Collection, error) {
	client, err := mongo.Connect(
		options.Client().
			ApplyURI(cfg.uri).
			SetServerSelectionTimeout(mongoPingTimeout).
			SetMonitor(otelmongo.NewMonitor()),
	)
	if err != nil {
		return nil, nil, err
	}

	pingCtx, cancel := context.WithTimeout(parent, mongoPingTimeout)
	err = client.Ping(pingCtx, nil)
	cancel()
	if err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, err
	}

	collection := client.Database(cfg.dbName).Collection(cfg.collName)
	idxCtx, cancel := context.WithTimeout(parent, mongoIndexTimeout)
	_, idxErr := collection.Indexes().CreateOne(idxCtx, mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	})
	cancel()
	if idxErr != nil {
		zap.S().Warnw("Could not create userId index (may already exist)", "error", idxErr)
	}

	return client, collection, nil
}

func mongoBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	// Exponential backoff with jitter, capped.
	backoff := mongoInitialBackoff * time.Duration(1<<(attempt-1))
	if backoff > mongoMaxBackoff {
		backoff = mongoMaxBackoff
	}

	// Add up to 100ms of jitter to avoid synchronized retries.
	jitter, err := cryptoJitter(100 * time.Millisecond)
	if err != nil {
		return backoff
	}

	backoff += jitter
	if backoff < 0 {
		return mongoInitialBackoff
	}
	return backoff
}

func cryptoJitter(maxDuration time.Duration) (time.Duration, error) {
	if maxDuration <= 0 {
		return 0, nil
	}

	limit := big.NewInt(int64(maxDuration))
	n, err := crand.Int(crand.Reader, limit)
	if err != nil {
		return 0, err
	}

	return time.Duration(n.Int64()), nil
}
