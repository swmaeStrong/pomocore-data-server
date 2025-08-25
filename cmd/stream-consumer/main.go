package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	categoryPatternService "pomocore-data/domains/categoryPattern/application/service"
	"pomocore-data/domains/patternClassifier/domain/core"
	mongoAdapter "pomocore-data/infrastructure/mongoDB/adapter"
	mongoConfig "pomocore-data/infrastructure/mongoDB/config"
	"pomocore-data/infrastructure/mongoDB/model"
	redisAdapter "pomocore-data/infrastructure/redis/adapter"
	"pomocore-data/infrastructure/redis/consumer"
	envConfig "pomocore-data/shared/common/config"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	envConfig.LoadEnv()

	// Initialize MongoDB
	mongoClient, err := mongoConfig.ConnectMongoDB()
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(envConfig.GetEnv("MONGO_DATABASE", "localhost:27017"))

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     envConfig.GetEnv("REDIS_ADDR", "localhost:6379"),
		Password: envConfig.GetEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize Pattern Classifier
	patternClassifier := core.NewPatternClassifier()
	if err := initializePatternClassifier(patternClassifier, db); err != nil {
		log.Fatalf("Failed to initialize pattern classifier: %v", err)
	}

	// Create MongoDB adapters
	categorizedDataRepo := mongoAdapter.NewCategorizedDataRepositoryPort(db)
	pomodoroUsageLogRepo := mongoAdapter.NewPomodoroUsageLogRepositoryPort(db)
	categoryPatternRepo := mongoAdapter.NewCategoryPatternRepositoryPort(db)

	// Create Redis adapters
	leaderboardCache := redisAdapter.NewLeaderboardCachePort(redisClient)
	classifierAdapter := redisAdapter.NewPatternClassifierAdapter(patternClassifier)

	// Create services
	categoryPatternUseCase := categoryPatternService.NewCategoryPatternService(categoryPatternRepo)

	// Create consumers
	pomodoroConsumer := consumer.NewPomodoroPatternConsumer(
		redisClient,
		classifierAdapter,
		categorizedDataRepo,
		pomodoroUsageLogRepo,
		categoryPatternUseCase,
		leaderboardCache,
	)

	// Leaderboard consumer is now optional since we update directly
	// leaderboardConsumer := consumer.NewLeaderboardConsumer(
	// 	redisClient,
	// 	leaderboardService,
	// )

	// Start consumers
	if err := pomodoroConsumer.Start(); err != nil {
		log.Fatalf("Failed to start pomodoro consumer: %v", err)
	}

	// Leaderboard consumer is no longer needed
	// if err := leaderboardConsumer.Start(); err != nil {
	// 	log.Fatalf("Failed to start leaderboard consumer: %v", err)
	// }

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	pomodoroConsumer.Stop()
	// leaderboardConsumer.Stop()
	log.Println("Shutdown complete")
}

func initializePatternClassifier(classifier *core.PatternClassifier, db *mongo.Database) error {
	collection := db.Collection("category_pattern")

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

	var patterns []model.CategoryPattern
	if err := cursor.All(context.Background(), &patterns); err != nil {
		return err
	}

	classifier.Initialize(patterns)
	log.Printf("Pattern classifier initialized with %d patterns", len(patterns))
	return nil
}
