package message

import (
	"time"
)

type SessionScoreMessage struct {
	UserID      string    `json:"userId"`
	SessionDate time.Time `json:"sessionDate"`
	Session     int       `json:"session"`
}

func NewSessionScoreMessage(userID string, sessionDate time.Time, session int) *SessionScoreMessage {
	return &SessionScoreMessage{
		UserID:      userID,
		SessionDate: sessionDate,
		Session:     session,
	}
}

// ToRedisValues converts the message to Redis stream values
func (m *SessionScoreMessage) ToRedisValues() map[string]interface{} {
	return map[string]interface{}{
		"userId":      m.UserID,
		"sessionDate": m.SessionDate.Format("2006-01-02"),
		"session":     m.Session,
	}
}
