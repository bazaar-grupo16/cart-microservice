package config

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestConnectDatabase(t *testing.T) {
	// Setup zap logger to avoid nil pointer panic
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	// Save original env vars
	origURI := os.Getenv("MONGO_URI")
	origDB := os.Getenv("MONGO_DB_NAME")
	origColl := os.Getenv("MONGO_COLLECTION")

	defer func() {
		os.Setenv("MONGO_URI", origURI)
		os.Setenv("MONGO_DB_NAME", origDB)
		os.Setenv("MONGO_COLLECTION", origColl)
	}()

	// Setup valid env vars for the test db
	os.Setenv("MONGO_URI", "mongodb://mongo-test:27017")
	os.Setenv("MONGO_DB_NAME", "cart_test_cfg")
	os.Setenv("MONGO_COLLECTION", "carts")

	// Since ConnectDatabase has zap.S().Fatal, we can't test failure easily without mocking os.Exit
	// We'll test the happy path here. In a real CI environment with `mongo-test`, it will succeed.
	// If it fails, the test binary will exit 1.

	// Ensure globals are nil
	setMongoGlobals(nil, nil)

	ConnectDatabase()

	if GetMongoClient() == nil {
		t.Errorf("expected MongoClient to be set")
	}
	if GetCartCollection() == nil {
		t.Errorf("expected CartCollection to be set")
	}
}

func TestSetMongoGlobals(t *testing.T) {
	setMongoGlobals(nil, nil)
	if GetMongoClient() != nil {
		t.Errorf("expected nil client")
	}
	if GetCartCollection() != nil {
		t.Errorf("expected nil collection")
	}
}
