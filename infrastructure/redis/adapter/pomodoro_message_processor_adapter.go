package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"pomocore-data/domains/message"
	pomodoroUseCase "pomocore-data/domains/pomodoro/application/usecase"
	"pomocore-data/infrastructure/redis/config"
	"pomocore-data/infrastructure/redis/consumer"
	"pomocore-data/shared/common/logger"
)

type PomodoroMessageProcessorAdapter struct {
	classifyUseCase pomodoroUseCase.ClassifyPomodoroUseCase
	redisClient     *redis.Client
}

func NewPomodoroMessageProcessorAdapter(
	classifyUseCase pomodoroUseCase.ClassifyPomodoroUseCase,
	redisClient *redis.Client,
) consumer.MessageProcessor {
	return &PomodoroMessageProcessorAdapter{
		classifyUseCase: classifyUseCase,
		redisClient:     redisClient,
	}
}

func (a *PomodoroMessageProcessorAdapter) ProcessBatch(ctx context.Context, messages []redis.XMessage) error {
	if len(messages) == 0 {
		return nil
	}

	// Parse messages from Redis format
	pomodoroMsgs, err := a.parseMessages(messages)
	if err != nil {
		return fmt.Errorf("failed to parse messages: %w", err)
	}

	if len(pomodoroMsgs) == 0 {
		return nil
	}

	// Process messages through use case
	_, sessionScoreMessages, err := a.classifyUseCase.Execute(ctx, pomodoroMsgs)
	if err != nil {
		// Log but continue - we want to acknowledge messages even if processing partially fails
		logger.Error("Error processing pomodoro messages", logger.WithError(err))
	}

	// Publish session score events
	if err := a.publishSessionScoreEvents(ctx, sessionScoreMessages); err != nil {
		logger.Error("Error publishing session score events", logger.WithError(err))
	}

	logger.Debug("Successfully processed batch of messages", zap.Int("batch_size", len(pomodoroMsgs)))
	return nil
}

func (a *PomodoroMessageProcessorAdapter) parseMessages(messages []redis.XMessage) ([]*message.PomodoroPatternClassifyMessage, error) {
	var pomodoroMsgs []*message.PomodoroPatternClassifyMessage

	for _, msg := range messages {
		pomodoroMsg, err := message.ParseFromRedisValues(msg.Values)
		if err != nil {
			logger.Warn("Error parsing message", zap.String("message_id", msg.ID), logger.WithError(err))
			// Skip invalid messages but continue processing
			continue
		}
		pomodoroMsgs = append(pomodoroMsgs, pomodoroMsg)
	}

	return pomodoroMsgs, nil
}

func (a *PomodoroMessageProcessorAdapter) publishSessionScoreEvents(ctx context.Context, sessionScoreMessages []*message.SessionScoreMessage) error {
	for _, sessionScoreMsg := range sessionScoreMessages {
		_, err := a.redisClient.XAdd(ctx, &redis.XAddArgs{
			Stream: config.SessionScoreSave.StreamKey,
			Values: sessionScoreMsg.ToRedisValues(),
		}).Result()

		if err != nil {
			logger.Error("Error sending sessionScore message",
				zap.String("user_id", sessionScoreMsg.UserID),
				zap.Int("session", sessionScoreMsg.Session),
				logger.WithError(err))
			continue
		}

		logger.Debug("SessionScore message sent",
			zap.String("user_id", sessionScoreMsg.UserID),
			zap.Int("session", sessionScoreMsg.Session))

		endedKey := a.getSessionStateKey(sessionScoreMsg.UserID, sessionScoreMsg.SessionDate, sessionScoreMsg.Session)
		if err := a.redisClient.Set(ctx, endedKey, "true", 10*time.Minute).Err(); err != nil {
			logger.Error("Error setting ended session key", zap.String("key", endedKey), logger.WithError(err))
		}
	}

	return nil
}

func (a *PomodoroMessageProcessorAdapter) getSessionStateKey(userID string, day time.Time, session int) string {
	dateStr := day.Format("2006-01-02")
	return fmt.Sprintf("session:processed:%s:%s:%d", userID, dateStr, session)
}
