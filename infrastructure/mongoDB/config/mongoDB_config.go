package config

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBConfig struct {
	URL      string
	Database string
}

func NewMongoDBConfig() *MongoDBConfig {
	envConfig.LoadEnv()
	return &MongoDBConfig{
		URL:      envConfig.GetEnv("MONGO_URI", "mongodb://localhost:27017"),
		Database: envConfig.GetEnv("MONGO_DATABASE", "pomocore"),
	}
}

func ConnectMongoDB() (*mongo.Client, error) {
	config := NewMongoDBConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URL))
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}
