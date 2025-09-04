package adapter

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"pomocore-data/domains/leaderboard/application/port"
	"pomocore-data/domains/leaderboard/domain"
)

type LeaderboardCacheAdapter struct {
	client    *redis.Client
	keyFormat string
}

func NewLeaderboardCachePort(client *redis.Client) port.LeaderboardCachePort {
	return &LeaderboardCacheAdapter{
		client:    client,
		keyFormat: "leaderboard:%s:%s",
	}
}

func (a *LeaderboardCacheAdapter) BatchIncreaseScore(ctx context.Context, entries []*domain.LeaderboardEntry) error {
	pipe := a.client.Pipeline()

	keyScores := make(map[string][]redis.Z)

	for _, entry := range entries {
		keys := entry.GetCategoryLeaderboardKeys()
		if entry.IsWorkCategory() {
			keys = append(keys, entry.GetWorkLeaderboardKeys()...)
		}

		for _, key := range keys {
			if keyScores[key] == nil {
				keyScores[key] = make([]redis.Z, 0)
			}

			keyScores[key] = append(keyScores[key], redis.Z{
				Score:  entry.Duration,
				Member: entry.UserID,
			})
		}
	}

	for key, scores := range keyScores {
		memberScores := make(map[string]float64)
		for _, score := range scores {
			memberScores[score.Member.(string)] += score.Score
		}

		for member, totalScore := range memberScores {
			pipe.ZIncrBy(ctx, key, totalScore, member)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute batch score increase: %w", err)
	}

	return nil
}
