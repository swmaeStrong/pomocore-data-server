package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CategoryPattern struct {
	ID             primitive.ObjectID `bson:"_id"`
	Category       string             `bson:"category"`
	AppPatterns    []string           `bson:"appPatterns"`
	DomainPatterns []string           `bson:"domainPatterns"`
}
