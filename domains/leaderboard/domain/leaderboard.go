package domain

import (
	"fmt"
	"time"
)

// LeaderboardResult represents a user's leaderboard information
type LeaderboardResult struct {
	UserID string
	Score  float64
	Rank   int64
}

// LeaderboardEntry represents a single entry for leaderboard operations
type LeaderboardEntry struct {
	UserID    string
	Category  string
	Duration  float64
	Timestamp float64
}

func NewLeaderboardEntry(userID, category string, duration, timestamp float64) *LeaderboardEntry {
	return &LeaderboardEntry{
		UserID:    userID,
		Category:  category,
		Duration:  duration,
		Timestamp: timestamp,
	}
}

var keyFormat = "leaderboard:%s:%s"

func (e *LeaderboardEntry) GetLeaderboardKey(periodType string) string {
	day := time.Unix(int64(e.Timestamp), 0)
	switch periodType {
	case "daily":
		return getDailyLeaderboardKey(e.Category, day)
	case "weekly":
		return getWeeklyLeaderboardKey(e.Category, day)
	case "monthly":
		return getMonthlyLeaderboardKey(e.Category, day)
	default:
		return getDailyLeaderboardKey(e.Category, day)
	}
}

func getDailyLeaderboardKey(category string, day time.Time) string {
	dateStr := day.Format("2006-01-02")
	return fmt.Sprintf(keyFormat, category, dateStr)
}

func getWeeklyLeaderboardKey(category string, day time.Time) string {
	year, week := day.ISOWeek()
	weekStr := fmt.Sprintf("%d-W%d", year, week)
	return fmt.Sprintf(keyFormat, category, weekStr)
}

func getMonthlyLeaderboardKey(category string, day time.Time) string {
	year := day.Year()
	month := int(day.Month())
	monthStr := fmt.Sprintf("%d-M%d", year, month)
	return fmt.Sprintf(keyFormat, category, monthStr)
}

func (e *LeaderboardEntry) getWorkLeaderboardKeys() []string {
	res := make([]string, 3)
	day := time.Unix(int64(e.Timestamp), 0)
	res[0] = getDailyLeaderboardKey("work", day)
	res[1] = getWeeklyLeaderboardKey("work", day)
	res[2] = getMonthlyLeaderboardKey("work", day)
	return res
}

func (e *LeaderboardEntry) getCategoryLeaderboardKeys() []string {
	res := make([]string, 3)
	day := time.Unix(int64(e.Timestamp), 0)
	res[0] = getDailyLeaderboardKey(e.Category, day)
	res[1] = getWeeklyLeaderboardKey(e.Category, day)
	res[2] = getMonthlyLeaderboardKey(e.Category, day)
	return res
}

func (e *LeaderboardEntry) GetLeaderboardKeys() []string {
	keys := e.getWorkLeaderboardKeys()
	keys = append(keys, e.getCategoryLeaderboardKeys()...)
	return keys
}
