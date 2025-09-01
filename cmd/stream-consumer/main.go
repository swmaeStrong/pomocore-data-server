package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	categoryPatternService "pomocore-data/domains/categoryPattern/application/service"
	"pomocore-data/domains/patternClassifier/domain/core"
	pomodoroService "pomocore-data/domains/pomodoro/application/service"
	mongoAdapter "pomocore-data/infrastructure/mongoDB/adapter"
	mongoConfig "pomocore-data/infrastructure/mongoDB/config"
	"pomocore-data/infrastructure/mongoDB/model"
	redisAdapter "pomocore-data/infrastructure/redis/adapter"
	redisConfig "pomocore-data/infrastructure/redis/config"
	"pomocore-data/infrastructure/redis/consumer"
	envConfig "pomocore-data/shared/common/config"
	"pomocore-data/shared/common/logger"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func main() {
	envConfig.LoadEnv()

	// Initialize logger
	if err := logger.InitFromEnv("pomocore-data"); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		logger.Fatal("Failed to load timezone", logger.WithError(err))
	}
	time.Local = loc

	// Initialize MongoDB
	mongoClient, err := mongoConfig.ConnectMongoDB()
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", logger.WithError(err))
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
		logger.Fatal("Failed to connect to Redis", logger.WithError(err))
	}

	// Initialize Pattern Classifier
	patternClassifier := core.NewPatternClassifier()
	if err := initializePatternClassifier(patternClassifier, db); err != nil {
		logger.Fatal("Failed to initialize pattern classifier", logger.WithError(err))
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

	// Create use case
	classifyUseCase := pomodoroService.NewPomodoroClassificationService(
		classifierAdapter,
		categorizedDataRepo,
		pomodoroUsageLogRepo,
		categoryPatternUseCase,
		leaderboardCache,
	)

	// Create message processor adapter
	messageProcessor := redisAdapter.NewPomodoroMessageProcessorAdapter(
		classifyUseCase,
		redisClient,
	)

	// Configure stream
	streamConfig := consumer.StreamConfig{
		StreamKey: redisConfig.PomodoroPatternMatch.StreamKey,
		Group:     redisConfig.PomodoroPatternMatch.Group,
		Consumer:  redisConfig.PomodoroPatternMatch.Consumer,
	}

	// Create abstract consumer with the processor
	pomodoroConsumer := consumer.NewAbstractConsumer(
		redisClient,
		streamConfig,
		messageProcessor,
		10,            // workerPool
		50,            // batchSize
		2*time.Second, // blockTime
	)

	if err := pomodoroConsumer.Start(); err != nil {
		logger.Fatal("Failed to start pomodoro consumer", logger.WithError(err))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
	pomodoroConsumer.Stop()
	logger.Info("Shutdown complete")
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
	logger.Info("Pattern classifier initialized", zap.Int("pattern_count", len(patterns)))
	return nil
}
