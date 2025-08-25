package consumer

import (
	"context"
	"errors"
	"fmt"
	"log"
	categoryPatternUseCase "pomocore-data/domains/categoryPattern/application/useCase"
	"pomocore-data/domains/leaderboard/domain"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"pomocore-data/domains/leaderboard/application/port"
	"pomocore-data/domains/message"
	pomodoroPort "pomocore-data/domains/pomodoro/application/port"
	"pomocore-data/infrastructure/redis/config"
)

type PomodoroPatternConsumer struct {
	client                 *redis.Client
	patternClassifier      PatternClassifier
	categorizedDataRepo    pomodoroPort.CategorizedDataRepositoryPort
	pomodoroUsageLogRepo   pomodoroPort.PomodoroUsageLogRepositoryPort
	categoryPatternUseCase categoryPatternUseCase.CategoryPatternUseCase
	leaderboardCache       port.LeaderboardCachePort
	workerPool             int
	batchSize              int
	ctx                    context.Context
	cancel                 context.CancelFunc
	wg                     sync.WaitGroup
	categoryToIdMap        map[string]primitive.ObjectID
}

type PatternClassifier interface {
	Classify(app, title, url string) (categoryID string, isLLMBased bool)
}

type ClassifyTask struct {
	Index int
	Msg   *message.PomodoroPatternClassifyMessage
}

type ClassifyResult struct {
	Index    int
	Category string
	IsLLM    bool
}

func NewPomodoroPatternConsumer(
	client *redis.Client,
	classifier PatternClassifier,
	categorizedDataRepo pomodoroPort.CategorizedDataRepositoryPort, // 나중에 어댑터로 받아주기
	pomodoroUsageLogRepo pomodoroPort.PomodoroUsageLogRepositoryPort, // 나중에 어댑터로 받아주기
	categoryPatternUseCase categoryPatternUseCase.CategoryPatternUseCase,
	leaderboardCache port.LeaderboardCachePort,
) *PomodoroPatternConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	categoryIdToCategoryMap, err := categoryPatternUseCase.GetCategoryToIdMap(ctx)
	if err != nil {
		categoryIdToCategoryMap = make(map[string]primitive.ObjectID)
		log.Printf("Warning: Failed to get category to ID map: %v", err)
	}
	fmt.Println("Category To IdMap: ", categoryIdToCategoryMap)
	return &PomodoroPatternConsumer{
		client:                 client,
		patternClassifier:      classifier,
		categorizedDataRepo:    categorizedDataRepo,
		pomodoroUsageLogRepo:   pomodoroUsageLogRepo,
		categoryPatternUseCase: categoryPatternUseCase,
		leaderboardCache:       leaderboardCache,
		workerPool:             10,
		batchSize:              50,
		ctx:                    ctx,
		cancel:                 cancel,
		categoryToIdMap:        categoryIdToCategoryMap,
	}
}

func (c *PomodoroPatternConsumer) Start() error {
	err := c.createConsumerGroup()
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	batchChan := make(chan []redis.XMessage, c.workerPool*2)
	fmt.Println(c.categoryToIdMap)
	for i := 0; i < c.workerPool; i++ {
		c.wg.Add(1)
		go c.batchWorker(i, batchChan)
	}

	c.wg.Add(1)
	go c.consume(batchChan)

	log.Printf("PomodoroPatternConsumer started with %d workers", c.workerPool)
	return nil
}

func (c *PomodoroPatternConsumer) Stop() {
	log.Println("Stopping PomodoroPatternConsumer...")
	c.cancel()
	c.wg.Wait()
	log.Println("PomodoroPatternConsumer stopped")
}

func (c *PomodoroPatternConsumer) createConsumerGroup() error {
	_, err := c.client.XGroupCreateMkStream(
		c.ctx,
		config.PomodoroPatternMatch.StreamKey,
		config.PomodoroPatternMatch.Group,
		"0",
	).Result()

	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (c *PomodoroPatternConsumer) consume(batchChan chan<- []redis.XMessage) {
	defer c.wg.Done()
	defer close(batchChan)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			messages, err := c.client.XReadGroup(c.ctx, &redis.XReadGroupArgs{
				Group:    config.PomodoroPatternMatch.Group,
				Consumer: config.PomodoroPatternMatch.Consumer,
				Streams:  []string{config.PomodoroPatternMatch.StreamKey, ">"},
				Count:    int64(c.batchSize),
				Block:    2 * time.Second,
			}).Result()

			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				log.Printf("Error reading from stream: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}

			var allMessages []redis.XMessage
			for _, stream := range messages {
				allMessages = append(allMessages, stream.Messages...)
			}

			if len(allMessages) > 0 {
				select {
				case batchChan <- allMessages:
					log.Printf("Sent batch of %d messages to workers", len(allMessages))
				case <-c.ctx.Done():
					return
				}
			}
		}
	}
}

func (c *PomodoroPatternConsumer) batchWorker(workerID int, batchChan <-chan []redis.XMessage) {
	defer c.wg.Done()
	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("Worker %d stopping", workerID)
			return
		case batch, ok := <-batchChan:
			if !ok {
				log.Printf("Worker %d: batch channel closed", workerID)
				return
			}
			log.Printf("Worker %d processing batch of %d messages", workerID, len(batch))
			c.processBatchMessages(batch)
		}
	}
}

func (c *PomodoroPatternConsumer) acknowledgeMessage(messageID string) {
	if err := c.client.XAck(
		c.ctx,
		config.PomodoroPatternMatch.StreamKey,
		config.PomodoroPatternMatch.Group,
		messageID,
	).Err(); err != nil {
		log.Printf("Error acknowledging message %s: %v", messageID, err)
	}
}

func (c *PomodoroPatternConsumer) classifyBatch(pomodoroMsgs []*message.PomodoroPatternClassifyMessage) []ClassifyResult {
	jobs := make(chan ClassifyTask, len(pomodoroMsgs))
	results := make(chan ClassifyResult, len(pomodoroMsgs))

	for w := 0; w < c.workerPool; w++ {
		go func() {
			for task := range jobs {
				category, isLLM := c.patternClassifier.Classify(
					task.Msg.App, task.Msg.Title, task.Msg.URL)
				results <- ClassifyResult{
					Index:    task.Index,
					Category: category,
					IsLLM:    isLLM,
				}
			}
		}()
	}

	for i, msg := range pomodoroMsgs {
		jobs <- ClassifyTask{Index: i, Msg: msg}
	}
	close(jobs)

	resultSlice := make([]ClassifyResult, len(pomodoroMsgs))
	for i := 0; i < len(pomodoroMsgs); i++ {
		result := <-results
		resultSlice[result.Index] = result
	}

	return resultSlice
}

func getSessionStateKey(userID string, day time.Time, session int) string {
	dateStr := day.Format("2006-01-02")
	return fmt.Sprintf("session:processed:%s:%s:%d", userID, dateStr, session)
}

func (c *PomodoroPatternConsumer) processBatchMessages(messages []redis.XMessage) {
	var err error
	if len(messages) == 0 {
		return
	}

	var pomodoroMsgs []*message.PomodoroPatternClassifyMessage
	var messageIDs []string

	var endedSessionMsgs []*message.SessionScoreMessage
	for _, msg := range messages {
		pomodoroMsg, err := message.ParseFromRedisValues(msg.Values)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			c.acknowledgeMessage(msg.ID)
			continue
		}
		pomodoroMsgs = append(pomodoroMsgs, pomodoroMsg)
		messageIDs = append(messageIDs, msg.ID)
		if pomodoroMsg.IsEnd {
			endedSessionMsgs = append(endedSessionMsgs, message.NewSessionScoreMessage(
				pomodoroMsg.UserID,
				pomodoroMsg.SessionDate,
				pomodoroMsg.Session,
			))
		}
	}

	if len(pomodoroMsgs) == 0 {
		return
	}

	classifyResults := c.classifyBatch(pomodoroMsgs)

	usageLogToCategoryIDMap := make(map[string]primitive.ObjectID)
	categorizedDataToCategoryIDMap := make(map[string]primitive.ObjectID)
	leaderboardUpdates := make([]*domain.LeaderboardEntry, len(pomodoroMsgs))

	for i, result := range classifyResults {
		pomodoroMsg := pomodoroMsgs[i]

		category := result.Category
		if category == "" {
			category = "Uncategorized"
			log.Printf("Classification failed for app=%s, title=%s, url=%s, using default category",
				pomodoroMsg.App, pomodoroMsg.Title, pomodoroMsg.URL)
		}

		leaderboardEntry := &domain.LeaderboardEntry{
			UserID:    pomodoroMsg.UserID,
			Category:  category,
			Duration:  pomodoroMsg.Duration,
			Timestamp: pomodoroMsg.Timestamp,
		}
		leaderboardUpdates[i] = leaderboardEntry

		categoryID := c.categoryToIdMap[category]
		if categoryID.IsZero() {
			log.Printf("No ObjectID found for category '%s', using zero ObjectID", category)
		}
		usageLogToCategoryIDMap[pomodoroMsg.PomodoroUsageLogID] = categoryID
		categorizedDataToCategoryIDMap[pomodoroMsg.CategorizedDataID] = categoryID
	}

	err = c.pomodoroUsageLogRepo.UpdateCategoryIDsBatch(c.ctx, usageLogToCategoryIDMap)
	if err != nil {
		log.Printf("Error updating usageLog data: %v", err)
		//TODO: 재시도
	}
	err = c.categorizedDataRepo.UpdateCategoryIDsBatch(c.ctx, categorizedDataToCategoryIDMap)
	if err != nil {
		log.Printf("Error updating categorized data: %v", err)
		//TODO: 재시도
	}

	err = c.leaderboardCache.BatchIncreaseScore(c.ctx, leaderboardUpdates)
	if err != nil {
		log.Printf("Error increasing score: %v", err)
		//TODO: 재시도
	}

	// Send sessionScore save messages for ended sessions
	for _, sessionScoreMsg := range endedSessionMsgs {
		_, err = c.client.XAdd(c.ctx, &redis.XAddArgs{
			Stream: config.SessionScoreSave.StreamKey,
			Values: sessionScoreMsg.ToRedisValues(),
		}).Result()

		if err != nil {
			log.Printf("Error sending sessionScore message for user %s, session %d: %v",
				sessionScoreMsg.UserID, sessionScoreMsg.Session, err)
		} else {
			log.Printf("SessionScore message sent for user %s, session %d",
				sessionScoreMsg.UserID, sessionScoreMsg.Session)
		}

		// Set ended session key to prevent duplicate processing
		endedKey := getSessionStateKey(sessionScoreMsg.UserID, sessionScoreMsg.SessionDate, sessionScoreMsg.Session)
		err := c.client.Set(c.ctx, endedKey, "true", 10*time.Minute).Err()
		if err != nil {
			log.Printf("Error setting ended session key %s: %v", endedKey, err)
		}
	}

	for _, messageID := range messageIDs {
		c.acknowledgeMessage(messageID)
	}

	log.Printf("Successfully processed batch of %d messages", len(pomodoroMsgs))
}
