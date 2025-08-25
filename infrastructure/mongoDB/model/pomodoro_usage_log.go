package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type PomodoroUsageLog struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	UserID            string             `bson:"userId"`
	CategorizedDataID primitive.ObjectID `bson:"categorizedDataId"`
	CategoryID        primitive.ObjectID `bson:"categoryId"`
	Session           int                `bson:"session"`
	SessionMinutes    int                `bson:"sessionMinutes"`
	SessionDate       time.Time          `bson:"sessionDate"`
	Timestamp         float64            `bson:"timestamp"`
	Duration          float64            `bson:"duration"`
}

func NewPomodoroUsageLog(
	userID string,
	categorizedDataID primitive.ObjectID,
	categoryID primitive.ObjectID,
	session int,
	sessionMinutes int,
	sessionDate time.Time,
	timestamp float64,
	duration float64,
) *PomodoroUsageLog {
	return &PomodoroUsageLog{
		UserID:            userID,
		CategorizedDataID: categorizedDataID,
		CategoryID:        categoryID,
		Session:           session,
		SessionMinutes:    sessionMinutes,
		SessionDate:       sessionDate,
		Timestamp:         timestamp,
		Duration:          duration,
	}
}

func (p *PomodoroUsageLog) UpdateCategoryID(categoryID primitive.ObjectID) {
	p.CategoryID = categoryID
}

func (p *PomodoroUsageLog) UpdateCategorizedDataID(categorizedDataID primitive.ObjectID) {
	p.CategorizedDataID = categorizedDataID
}
