package adapter

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
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
		log.Printf("Error in FindAll: %v", err)
		return nil, err
	}
	log.Printf("FindAll returned %d patterns", len(categoryMap))
	categoryMapByCategory := make(map[string]primitive.ObjectID)
	for _, pattern := range categoryMap {
		log.Printf("Adding pattern: Category=%s, ID=%s", pattern.Category, pattern.ID.Hex())
		categoryMapByCategory[pattern.Category] = pattern.ID
	}
	log.Printf("Final categoryMapByCategory size: %d", len(categoryMapByCategory))
	return categoryMapByCategory, nil
}

func (a *CategoryPatternRepositoryAdapter) FindAll(ctx context.Context) ([]model.CategoryPattern, error) {
	var categoryPatterns []model.CategoryPattern

	log.Printf("Finding all documents from collection: %s", a.collection.Name())
	cursor, err := a.collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Error finding documents: %v", err)
		return nil, err
	}

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("Error closing cursor: %v", err)
		}
	}()

	if err := cursor.All(ctx, &categoryPatterns); err != nil {
		log.Printf("Error decoding cursor: %v", err)
		return nil, err
	}

	log.Printf("Found %d categoryPatterns", len(categoryPatterns))
	for i, pattern := range categoryPatterns {
		log.Printf("Pattern %d: ID=%s, Category=%s", i, pattern.ID.Hex(), pattern.Category)
	}

	return categoryPatterns, nil
}
