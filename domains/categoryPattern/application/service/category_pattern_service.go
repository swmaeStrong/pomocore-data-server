package service

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"pomocore-data/domains/categoryPattern/application/port"
	"pomocore-data/domains/categoryPattern/application/useCase"
)

type CategoryPatternService struct {
	repo port.CategoryPatternRepositoryPort
}

func NewCategoryPatternService(repo port.CategoryPatternRepositoryPort) useCase.CategoryPatternUseCase {
	return &CategoryPatternService{
		repo: repo,
	}
}

func (s *CategoryPatternService) GetCategoryToIdMap(ctx context.Context) (map[string]primitive.ObjectID, error) {
	return s.repo.FindCategoryToIdMap(ctx)
}

func (s *CategoryPatternService) GetIdToCategoryMap(ctx context.Context) (map[string]string, error) {
	return s.repo.FindIdToCategoryMap(ctx)
}
