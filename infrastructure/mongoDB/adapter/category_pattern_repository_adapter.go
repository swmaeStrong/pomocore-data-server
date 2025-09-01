package adapter

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	categoryPatternPort "pomocore-data/domains/categoryPattern/application/port"
	"pomocore-data/infrastructure/mongoDB/model"
)

type CategoryPatternRepositoryAdapter struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewCategoryPatternRepositoryPort(db *mongo.Database) categoryPatternPort.CategoryPatternRepositoryPort {
	return &CategoryPatternRepositoryAdapter{
		db:         db,
		collection: db.Collection("category_pattern"),
	}
}

func (a *CategoryPatternRepositoryAdapter) FindAllCategories(ctx context.Context) ([]string, error) {
	categoryPatterns, err := a.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var categories []string
	for _, pattern := range categoryPatterns {
		categories = append(categories, pattern.Category)
	}

	return categories, nil
}

func (a *CategoryPatternRepositoryAdapter) FindIdToCategoryMap(cxt context.Context) (map[string]string, error) {
	categoryPatterns, err := a.FindAll(cxt)
	if err != nil {
		return nil, err
	}

	categoryMap := make(map[string]string)
	for _, pattern := range categoryPatterns {
		categoryMap[pattern.ID.Hex()] = pattern.Category
	}
	return categoryMap, nil
}

func (a *CategoryPatternRepositoryAdapter) FindCategoryToIdMap(cxt context.Context) (map[string]primitive.ObjectID, error) {
	categoryMap, err := a.FindAll(cxt)
	if err != nil {
		return nil, err
	}
	categoryMapByCategory := make(map[string]primitive.ObjectID)
	for _, pattern := range categoryMap {
		categoryMapByCategory[pattern.Category] = pattern.ID
	}
	return categoryMapByCategory, nil
}

func (a *CategoryPatternRepositoryAdapter) FindAll(ctx context.Context) ([]model.CategoryPattern, error) {
	var categoryPatterns []model.CategoryPattern

	cursor, err := a.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := cursor.Close(ctx); err != nil {
		}
	}()

	if err := cursor.All(ctx, &categoryPatterns); err != nil {
		return nil, err
	}

	return categoryPatterns, nil
}
