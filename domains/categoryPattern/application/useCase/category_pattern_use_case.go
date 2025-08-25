package useCase

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CategoryPatternUseCase interface {
	GetCategoryToIdMap(ctx context.Context) (map[string]primitive.ObjectID, error)
	GetIdToCategoryMap(ctx context.Context) (map[string]string, error)
}
