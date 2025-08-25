package port

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"pomocore-data/infrastructure/mongoDB/model"
)

type AppUrlTitleKey struct {
	App   string
	URL   string
	Title string
}

type CategorizedDataRepositoryPort interface {
	Save(ctx context.Context, data *model.CategorizedData) (*primitive.ObjectID, error)
	FindByAppUrlTitle(ctx context.Context, app, url, title string) (*model.CategorizedData, error)
	UpdateCategoryID(ctx context.Context, id primitive.ObjectID, categoryID primitive.ObjectID) error

	SaveBatch(ctx context.Context, dataList []*model.CategorizedData) ([]*primitive.ObjectID, error)
	UpdateCategoryIDsBatch(ctx context.Context, categorizedDataToCategoryIDMap map[string]primitive.ObjectID) error
}
