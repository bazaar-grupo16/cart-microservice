package config

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

// MongoClient is the shared MongoDB client used across the application.
var MongoClient *mongo.Client

// CartCollection is the MongoDB collection that stores cart documents.
var CartCollection *mongo.Collection

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		zap.S().Fatalw("Failed to create MongoDB client", "error", err)
	}

	// Verify the connection is reachable before proceeding.
	if err := client.Ping(ctx, nil); err != nil {
		zap.S().Fatalw("Failed to ping MongoDB", "error", err)
	}

	MongoClient = client
	CartCollection = client.Database(dbName).Collection(collName)

	// Ensure a unique sparse index on userId so each user has at most one cart document.
	_, err = CartCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	})
	if err != nil {
		zap.S().Warnw("Could not create userId index (may already exist)", "error", err)
	}

	zap.S().Infow("Connected to MongoDB", "db", dbName, "collection", collName)
}
