package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CategorizedData struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	App        string             `bson:"app"`
	URL        string             `bson:"url"`
	Title      string             `bson:"title"`
	CategoryID primitive.ObjectID `bson:"categoryId"`
	IsLLMBased bool               `bson:"isLLMBased"`
}

func NewCategorizedData(app, url, title string, categoryID primitive.ObjectID, isLLMBased bool) *CategorizedData {
	return &CategorizedData{
		ID:         primitive.NewObjectID(),
		App:        app,
		URL:        url,
		Title:      title,
		CategoryID: categoryID,
		IsLLMBased: isLLMBased,
	}
}

func (c *CategorizedData) UpdateCategoryID(categoryID primitive.ObjectID) {
	c.CategoryID = categoryID
}

func (c *CategorizedData) CheckLLMBased(isLLMBased bool) {
	c.IsLLMBased = isLLMBased
}
