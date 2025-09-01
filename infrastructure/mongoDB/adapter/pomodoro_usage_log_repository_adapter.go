package adapter

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	pomodoroPort "pomocore-data/domains/pomodoro/application/port"
	"pomocore-data/infrastructure/mongoDB/model"
	"pomocore-data/shared/common/logger"
	"time"

	"go.uber.org/zap"
)

type PomodoroUsageLogRepositoryAdapter struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewPomodoroUsageLogRepositoryPort(db *mongo.Database) pomodoroPort.PomodoroUsageLogRepositoryPort {
	return &PomodoroUsageLogRepositoryAdapter{
		db:         db,
		collection: db.Collection("pomodoro_usage_log"),
	}
}

func (a *PomodoroUsageLogRepositoryAdapter) Save(ctx context.Context, logData *model.PomodoroUsageLog) (*primitive.ObjectID, error) {
	if logData.ID.IsZero() {
		logData.ID = primitive.NewObjectID()
	}

	_, err := a.collection.InsertOne(ctx, logData)
	if err != nil {
		return nil, err
	}

	return &logData.ID, nil
}

func (a *PomodoroUsageLogRepositoryAdapter) FindByUserIDAndSession(ctx context.Context, userID string, sessionDate time.Time, session int) (*model.PomodoroUsageLog, error) {
	var result model.PomodoroUsageLog

	filter := bson.M{
		"userId":      userID,
		"sessionDate": sessionDate,
		"session":     session,
	}

	err := a.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Not found
		}
		return nil, err
	}

	return &result, nil
}

func (a *PomodoroUsageLogRepositoryAdapter) UpdateCategoryID(ctx context.Context, id primitive.ObjectID, categoryID primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"categoryId": categoryID}}

	result, err := a.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		logger.Debug("No document found with id", zap.String("id", id.Hex()))
	}

	return nil
}

func (a *PomodoroUsageLogRepositoryAdapter) UpdateCategorizedDataID(ctx context.Context, id primitive.ObjectID, categorizedDataID primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"categorizedDataId": categorizedDataID}}

	result, err := a.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		logger.Debug("No document found with id", zap.String("id", id.Hex()))
	}

	return nil
}

func (a *PomodoroUsageLogRepositoryAdapter) SaveBatch(ctx context.Context, logs []*model.PomodoroUsageLog) ([]*primitive.ObjectID, error) {
	if len(logs) == 0 {
		return []*primitive.ObjectID{}, nil
	}

	// Prepare documents for insertion
	docs := make([]interface{}, 0, len(logs))
	ids := make([]*primitive.ObjectID, 0, len(logs))

	for _, logData := range logs {
		if logData.ID.IsZero() {
			logData.ID = primitive.NewObjectID()
		}
		docs = append(docs, logData)
		ids = append(ids, &logData.ID)
	}

	_, err := a.collection.InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (a *PomodoroUsageLogRepositoryAdapter) UpdateCategorizedDataIDsBatch(ctx context.Context, usageLogToCategorizedDataMap map[string]primitive.ObjectID) error {
	if len(usageLogToCategorizedDataMap) == 0 {
		return nil
	}

	var operations []mongo.WriteModel

	for usageLogIDStr, categorizedDataID := range usageLogToCategorizedDataMap {
		usageLogID, err := primitive.ObjectIDFromHex(usageLogIDStr)
		if err != nil {
			logger.Warn("Invalid ObjectID format",
				zap.String("usage_log_id", usageLogIDStr),
				logger.WithError(err))
			continue
		}

		filter := bson.M{"_id": usageLogID}
		update := bson.M{"$set": bson.M{"categorizedDataId": categorizedDataID}}

		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(filter)
		operation.SetUpdate(update)
		operations = append(operations, operation)
	}

	if len(operations) == 0 {
		return nil
	}

	result, err := a.collection.BulkWrite(ctx, operations)
	if err != nil {
		return err
	}

	logger.Debug("Updated pomodoro usage logs with categorized data IDs",
		zap.Int64("modified_count", result.ModifiedCount))
	return nil
}

func (a *PomodoroUsageLogRepositoryAdapter) UpdateCategoryIDsBatch(ctx context.Context, usageLogToCategoryIDMap map[string]primitive.ObjectID) error {
	if len(usageLogToCategoryIDMap) == 0 {
		return nil
	}

	var operations []mongo.WriteModel

	for usageLogIDStr, categoryID := range usageLogToCategoryIDMap {
		usageLogID, err := primitive.ObjectIDFromHex(usageLogIDStr)
		if err != nil {
			logger.Warn("Invalid ObjectID format",
				zap.String("usage_log_id", usageLogIDStr),
				logger.WithError(err))
			continue
		}

		filter := bson.M{"_id": usageLogID}
		update := bson.M{"$set": bson.M{"categoryId": categoryID}}

		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(filter)
		operation.SetUpdate(update)
		operations = append(operations, operation)
	}

	if len(operations) == 0 {
		return nil
	}

	result, err := a.collection.BulkWrite(ctx, operations)
	if err != nil {
		return err
	}

	logger.Debug("Updated pomodoro usage logs with category IDs",
		zap.Int64("modified_count", result.ModifiedCount))
	return nil
}
