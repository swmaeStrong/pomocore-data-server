package usecase

import (
	"context"

	"pomocore-data/domains/leaderboard/domain"
	"pomocore-data/domains/message"
)

type ClassifyPomodoroUseCase interface {
	Execute(
		ctx context.Context,
		pomodoroMsgs []*message.PomodoroPatternClassifyMessage,
	) ([]*domain.LeaderboardEntry, []*message.SessionScoreMessage, error)
	
	RefreshCategoryMapping(ctx context.Context) error
}