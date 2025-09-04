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

func (e *LeaderboardEntry) GetWorkLeaderboardKeys() []string {
	res := make([]string, 3)
	day := time.Unix(int64(e.Timestamp), 0)
	res[0] = getDailyLeaderboardKey("work", day)
	res[1] = getWeeklyLeaderboardKey("work", day)
	res[2] = getMonthlyLeaderboardKey("work", day)
	return res
}

func (e *LeaderboardEntry) GetCategoryLeaderboardKeys() []string {
	res := make([]string, 3)
	day := time.Unix(int64(e.Timestamp), 0)
	res[0] = getDailyLeaderboardKey(e.Category, day)
	res[1] = getWeeklyLeaderboardKey(e.Category, day)
	res[2] = getMonthlyLeaderboardKey(e.Category, day)
	return res
}

func (e *LeaderboardEntry) IsWorkCategory() bool {
	workCategory := make(map[string]bool)
	workCategory["Development"] = true
	workCategory["LLM"] = true
	workCategory["Documentation"] = true
	workCategory["Design"] = true
	workCategory["Video Editing"] = true
	workCategory["Education"] = true
	workCategory["Productivity"] = true
	workCategory["Finance"] = true
	workCategory["File Management"] = true
	workCategory["Browsing"] = true
	workCategory["Marketing"] = true
	workCategory["System & Utilities"] = true
	workCategory["Meetings"] = true
	return workCategory[e.Category] == true
}
