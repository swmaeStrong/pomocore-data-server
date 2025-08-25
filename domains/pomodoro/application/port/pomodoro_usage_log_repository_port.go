package port

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"pomocore-data/infrastructure/mongoDB/model"
	"time"
)

type PomodoroUsageLogRepositoryPort interface {
	Save(ctx context.Context, log *model.PomodoroUsageLog) (*primitive.ObjectID, error)
	FindByUserIDAndSession(ctx context.Context, userID string, sessionDate time.Time, session int) (*model.PomodoroUsageLog, error)
	UpdateCategoryID(ctx context.Context, id primitive.ObjectID, categoryID primitive.ObjectID) error
	UpdateCategorizedDataID(ctx context.Context, id primitive.ObjectID, categorizedDataID primitive.ObjectID) error

	// Batch operations for N+1 optimization
	SaveBatch(ctx context.Context, logs []*model.PomodoroUsageLog) ([]*primitive.ObjectID, error)
	UpdateCategorizedDataIDsBatch(ctx context.Context, usageLogToCategorizedDataMap map[string]primitive.ObjectID) error
	UpdateCategoryIDsBatch(ctx context.Context, usageLogToCategoryIDMap map[string]primitive.ObjectID) error
}
