package adapter

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	pomodoroPort "pomocore-data/domains/pomodoro/application/port"
	"pomocore-data/infrastructure/mongoDB/model"
)

type CategorizedDataRepositoryAdapter struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewCategorizedDataRepositoryPort(db *mongo.Database) pomodoroPort.CategorizedDataRepositoryPort {
	return &CategorizedDataRepositoryAdapter{
		db:         db,
		collection: db.Collection("categorized_data"),
	}
}

func (a *CategorizedDataRepositoryAdapter) Save(ctx context.Context, data *model.CategorizedData) (*primitive.ObjectID, error) {
	if data.ID.IsZero() {
		data.ID = primitive.NewObjectID()
	}

	_, err := a.collection.InsertOne(ctx, data)
	if err != nil {
		return nil, err
	}

	return &data.ID, nil
}

func (a *CategorizedDataRepositoryAdapter) FindByAppUrlTitle(ctx context.Context, app, url, title string) (*model.CategorizedData, error) {
	var result model.CategorizedData

	filter := bson.M{
		"app":   app,
		"url":   url,
		"title": title,
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

func (a *CategorizedDataRepositoryAdapter) UpdateCategoryID(ctx context.Context, id primitive.ObjectID, categoryID primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"categoryId": categoryID}}

	result, err := a.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		log.Printf("No document found with id: %s", id.Hex())
	}

	return nil
}

func (a *CategorizedDataRepositoryAdapter) FindManyByAppUrlTitleBatch(ctx context.Context, keys []pomodoroPort.AppUrlTitleKey) (map[pomodoroPort.AppUrlTitleKey]*model.CategorizedData, error) {
	if len(keys) == 0 {
		return make(map[pomodoroPort.AppUrlTitleKey]*model.CategorizedData), nil
	}

	// Build OR query for all keys
	orConditions := make([]bson.M, 0, len(keys))
	for _, key := range keys {
		orConditions = append(orConditions, bson.M{
			"app":   key.App,
			"url":   key.URL,
			"title": key.Title,
		})
	}

	filter := bson.M{"$or": orConditions}

	cursor, err := a.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[pomodoroPort.AppUrlTitleKey]*model.CategorizedData)

	for cursor.Next(ctx) {
		var data model.CategorizedData
		if err := cursor.Decode(&data); err != nil {
			return nil, err
		}

		key := pomodoroPort.AppUrlTitleKey{
			App:   data.App,
			URL:   data.URL,
			Title: data.Title,
		}
		result[key] = &data
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (a *CategorizedDataRepositoryAdapter) SaveBatch(ctx context.Context, dataList []*model.CategorizedData) ([]*primitive.ObjectID, error) {
	if len(dataList) == 0 {
		return []*primitive.ObjectID{}, nil
	}

	// Prepare documents for insertion
	docs := make([]interface{}, 0, len(dataList))
	ids := make([]*primitive.ObjectID, 0, len(dataList))

	for _, data := range dataList {
		if data.ID.IsZero() {
			data.ID = primitive.NewObjectID()
		}
		docs = append(docs, data)
		ids = append(ids, &data.ID)
	}

	_, err := a.collection.InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (a *CategorizedDataRepositoryAdapter) UpdateCategoryIDsBatch(ctx context.Context, categorizedDataToCategoryIDMap map[string]primitive.ObjectID) error {
	if len(categorizedDataToCategoryIDMap) == 0 {
		return nil
	}

	var operations []mongo.WriteModel

	for categorizedDataIDStr, categoryID := range categorizedDataToCategoryIDMap {
		categorizedDataID, err := primitive.ObjectIDFromHex(categorizedDataIDStr)
		if err != nil {
			log.Printf("Invalid ObjectID format: %s, error: %v", categorizedDataIDStr, err)
			continue
		}

		filter := bson.M{"_id": categorizedDataID}
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

	log.Printf("Updated %d categorized data with category IDs", result.ModifiedCount)
	return nil
}
