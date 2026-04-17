package config

import (
	"context"
	"os"
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

// MongoClient is the shared MongoDB client used across the application.
var MongoClient *mongo.Client

// CartCollection is the MongoDB collection that stores cart documents.
var CartCollection *mongo.Collection

var (
	connMu sync.RWMutex

	lastMongoCfg atomic.Value // mongoCfg
)

type mongoCfg struct {
	uri      string
	dbName   string
	collName string
}

// GetMongoClient returns the current Mongo client.
// It may temporarily be nil during startup or a reconnect attempt.
func GetMongoClient() *mongo.Client {
	connMu.RLock()
	defer connMu.RUnlock()
	return MongoClient
}

// GetCartCollection returns the current carts collection.
// It may temporarily be nil during startup or a reconnect attempt.
func GetCartCollection() *mongo.Collection {
	connMu.RLock()
	defer connMu.RUnlock()
	return CartCollection
}

func setMongoGlobals(client *mongo.Client, collection *mongo.Collection) {
	connMu.Lock()
	defer connMu.Unlock()
	MongoClient = client
	CartCollection = collection
}

// ConnectDatabase establishes a connection to MongoDB using the MONGO_URI
// environment variable and stores the cart collection in CartCollection.
// It also ensures a unique index on the "userId" field so that one document
// per user is enforced at the database level.
func ConnectDatabase() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		zap.S().Fatal("MONGO_URI environment variable is not set")
	}

	dbName := os.Getenv("MONGO_DB_NAME")
	if dbName == "" {
		zap.S().Fatal("MONGO_DB_NAME environment variable is not set")
	}

	collName := os.Getenv("MONGO_COLLECTION")
	if collName == "" {
		zap.S().Fatal("MONGO_COLLECTION environment variable is not set")
	}

	cfg := mongoCfg{uri: uri, dbName: dbName, collName: collName}
	lastMongoCfg.Store(cfg)

	client, collection, err := connectMongoWithRetry(context.Background(), cfg, mongoRetryTimeout)
	if err != nil {
		zap.S().Fatalw("MongoDB connection failed after retries", "error", err)
	}

	setMongoGlobals(client, collection)
	startMongoReconnectMonitor()

	zap.S().Infow("Connected to MongoDB", "db", dbName, "collection", collName)
}
