package message

import (
	"strconv"
	"time"
)

type PomodoroPatternClassifyMessage struct {
	UserID             string    `json:"userId"`
	CategorizedDataID  string    `json:"categorizedDataId"`
	PomodoroUsageLogID string    `json:"pomodoroUsageLogId"`
	URL                string    `json:"url"`
	Title              string    `json:"title"`
	App                string    `json:"app"`
	Session            int       `json:"session"`
	SessionDate        time.Time `json:"sessionDate"`
	SessionMinutes     int       `json:"sessionMinutes"`
	Duration           float64   `json:"duration"`
	Timestamp          float64   `json:"timestamp"`
	IsEnd              bool      `json:"isEnd"`
}

// ParseFromRedisValues creates a message from Redis stream values (all strings)
func ParseFromRedisValues(values map[string]interface{}) (*PomodoroPatternClassifyMessage, error) {
	msg := &PomodoroPatternClassifyMessage{}

	// Helper function to get string value
	getString := func(key string) string {
		if v, ok := values[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	msg.UserID = getString("userId")
	msg.CategorizedDataID = getString("categorizedDataId")
	msg.PomodoroUsageLogID = getString("pomodoroUsageLogId")
	msg.URL = getString("url")
	msg.Title = getString("title")
	msg.App = getString("app")

	// Parse integer fields
	if s := getString("session"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			msg.Session = v
		}
	}

	if s := getString("sessionMinutes"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			msg.SessionMinutes = v
		}
	}

	// Parse date field
	if s := getString("sessionDate"); s != "" {
		// Try date-only format first
		if t, err := time.Parse("2006-01-02", s); err == nil {
			msg.SessionDate = t
		} else if t, err := time.Parse(time.RFC3339, s); err == nil {
			msg.SessionDate = t
		}
	}

	// Parse float fields
	if s := getString("duration"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			msg.Duration = v
		}
	}

	if s := getString("timestamp"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			msg.Timestamp = v
		}
	}

	// Parse bool field
	if s := getString("isEnd"); s != "" {
		if v, err := strconv.ParseBool(s); err == nil {
			msg.IsEnd = v
		}
	}

	return msg, nil
}
