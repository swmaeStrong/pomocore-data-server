package port

import "context"

type PatternClassifierPort interface {
	GetIdToCategoryMap(cxt context.Context) (map[string]string, error)
	GetCategoryToIdMap(cxt context.Context) (map[string]string, error)
}
