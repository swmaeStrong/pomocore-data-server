package port

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CategoryPatternRepositoryPort interface {
	FindAllCategories(cxt context.Context) ([]string, error)
	FindCategoryToIdMap(cxt context.Context) (map[string]primitive.ObjectID, error)
	FindIdToCategoryMap(cxt context.Context) (map[string]string, error)
}
