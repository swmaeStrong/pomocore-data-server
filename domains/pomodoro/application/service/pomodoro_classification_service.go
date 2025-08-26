package usecase

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"

	categoryPatternUseCase "pomocore-data/domains/categoryPattern/application/useCase"
	"pomocore-data/domains/leaderboard/application/port"
	"pomocore-data/domains/leaderboard/domain"
	"pomocore-data/domains/message"
	pomodoroPort "pomocore-data/domains/pomodoro/application/port"
	pomodoroUseCase "pomocore-data/domains/pomodoro/application/usecase"
)

type PatternClassifier interface {
	Classify(app, title, url string) (categoryID string, isLLMBased bool)
}

type ClassificationTask struct {
	Index int
	Msg   *message.PomodoroPatternClassifyMessage
}

type ClassificationResult struct {
	Index    int
	Category string
	IsLLM    bool
}

type PomodoroClassificationService struct {
	patternClassifier      PatternClassifier
	categorizedDataRepo    pomodoroPort.CategorizedDataRepositoryPort
	pomodoroUsageLogRepo   pomodoroPort.PomodoroUsageLogRepositoryPort
	categoryPatternUseCase categoryPatternUseCase.CategoryPatternUseCase
	leaderboardCache       port.LeaderboardCachePort
	categoryToIdMap        map[string]primitive.ObjectID
	workerPool             int
	mu                     sync.RWMutex
}

func NewPomodoroClassificationService(
	patternClassifier PatternClassifier,
	categorizedDataRepo pomodoroPort.CategorizedDataRepositoryPort,
	pomodoroUsageLogRepo pomodoroPort.PomodoroUsageLogRepositoryPort,
	categoryPatternUseCase categoryPatternUseCase.CategoryPatternUseCase,
	leaderboardCache port.LeaderboardCachePort,
) pomodoroUseCase.ClassifyPomodoroUseCase {
	ctx := context.Background()
	categoryIdToCategoryMap, err := categoryPatternUseCase.GetCategoryToIdMap(ctx)
	if err != nil {
		categoryIdToCategoryMap = make(map[string]primitive.ObjectID)
		log.Printf("Warning: Failed to get category to ID map: %v", err)
	}
	fmt.Println("Category To IdMap: ", categoryIdToCategoryMap)

	return &PomodoroClassificationService{
		patternClassifier:      patternClassifier,
		categorizedDataRepo:    categorizedDataRepo,
		pomodoroUsageLogRepo:   pomodoroUsageLogRepo,
		categoryPatternUseCase: categoryPatternUseCase,
		leaderboardCache:       leaderboardCache,
		categoryToIdMap:        categoryIdToCategoryMap,
		workerPool:             10,
	}
}

// Execute processes a batch of pomodoro messages and returns classification results
func (s *PomodoroClassificationService) Execute(
	ctx context.Context,
	pomodoroMsgs []*message.PomodoroPatternClassifyMessage,
) ([]*domain.LeaderboardEntry, []*message.SessionScoreMessage, error) {
	if len(pomodoroMsgs) == 0 {
		return nil, nil, nil
	}

	// Classify messages
	classificationResults := s.classifyBatch(pomodoroMsgs)

	// Prepare data for updates
	usageLogToCategoryIDMap := make(map[string]primitive.ObjectID)
	categorizedDataToCategoryIDMap := make(map[string]primitive.ObjectID)
	leaderboardUpdates := make([]*domain.LeaderboardEntry, 0, len(pomodoroMsgs))
	sessionScoreMessages := make([]*message.SessionScoreMessage, 0)

	for i, result := range classificationResults {
		pomodoroMsg := pomodoroMsgs[i]

		category := result.Category
		if category == "" {
			category = "Uncategorized"
			log.Printf("Classification failed for app=%s, title=%s, url=%s, using default category",
				pomodoroMsg.App, pomodoroMsg.Title, pomodoroMsg.URL)
		}

		// Create leaderboard entry
		leaderboardEntry := domain.NewLeaderboardEntry(
			pomodoroMsg.UserID,
			category,
			pomodoroMsg.Duration,
			pomodoroMsg.Timestamp,
		)
		leaderboardUpdates = append(leaderboardUpdates, leaderboardEntry)

		// Map category to ObjectID
		categoryID := s.getCategoryID(category)
		if categoryID.IsZero() {
			log.Printf("No ObjectID found for category '%s', using zero ObjectID", category)
		}
		usageLogToCategoryIDMap[pomodoroMsg.PomodoroUsageLogID] = categoryID
		categorizedDataToCategoryIDMap[pomodoroMsg.CategorizedDataID] = categoryID

		// Collect ended session messages
		if pomodoroMsg.IsEnd {
			sessionScoreMessages = append(sessionScoreMessages, message.NewSessionScoreMessage(
				pomodoroMsg.UserID,
				pomodoroMsg.SessionDate,
				pomodoroMsg.Session,
			))
		}
	}

	// Update repositories
	if err := s.pomodoroUsageLogRepo.UpdateCategoryIDsBatch(ctx, usageLogToCategoryIDMap); err != nil {
		log.Printf("Error updating usageLog data: %v", err)
		// Continue processing despite error
	}

	if err := s.categorizedDataRepo.UpdateCategoryIDsBatch(ctx, categorizedDataToCategoryIDMap); err != nil {
		log.Printf("Error updating categorized data: %v", err)
		// Continue processing despite error
	}

	// Update leaderboard cache
	if err := s.leaderboardCache.BatchIncreaseScore(ctx, leaderboardUpdates); err != nil {
		log.Printf("Error increasing score: %v", err)
		// Continue processing despite error
	}

	return leaderboardUpdates, sessionScoreMessages, nil
}

// classifyBatch classifies a batch of pomodoro messages using parallel workers
func (s *PomodoroClassificationService) classifyBatch(pomodoroMsgs []*message.PomodoroPatternClassifyMessage) []ClassificationResult {
	jobs := make(chan ClassificationTask, len(pomodoroMsgs))
	results := make(chan ClassificationResult, len(pomodoroMsgs))

	// Start workers
	for w := 0; w < s.workerPool; w++ {
		go func() {
			for task := range jobs {
				category, isLLM := s.patternClassifier.Classify(
					task.Msg.App, task.Msg.Title, task.Msg.URL)
				results <- ClassificationResult{
					Index:    task.Index,
					Category: category,
					IsLLM:    isLLM,
				}
			}
		}()
	}

	// Send jobs
	for i, msg := range pomodoroMsgs {
		jobs <- ClassificationTask{Index: i, Msg: msg}
	}
	close(jobs)

	// Collect results
	resultSlice := make([]ClassificationResult, len(pomodoroMsgs))
	for i := 0; i < len(pomodoroMsgs); i++ {
		result := <-results
		resultSlice[result.Index] = result
	}

	return resultSlice
}

// getCategoryID returns the ObjectID for a given category name
func (s *PomodoroClassificationService) getCategoryID(category string) primitive.ObjectID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.categoryToIdMap[category]
}

// RefreshCategoryMapping refreshes the category to ID mapping from the database
func (s *PomodoroClassificationService) RefreshCategoryMapping(ctx context.Context) error {
	categoryIdToCategoryMap, err := s.categoryPatternUseCase.GetCategoryToIdMap(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.categoryToIdMap = categoryIdToCategoryMap
	s.mu.Unlock()

	log.Printf("Refreshed category to ID map with %d categories", len(categoryIdToCategoryMap))
	return nil
}
