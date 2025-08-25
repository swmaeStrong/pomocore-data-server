package port

import (
	"context"
	"pomocore-data/domains/leaderboard/domain"
)

type LeaderboardCachePort interface {

	// BatchIncreaseScore increases multiple users' scores in batch
	BatchIncreaseScore(ctx context.Context, entries []*domain.LeaderboardEntry) error
}
