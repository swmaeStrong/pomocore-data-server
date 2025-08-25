package adapter

import (
	"pomocore-data/domains/patternClassifier/domain/core"
)

type PatternClassifierAdapter struct {
	classifier *core.PatternClassifier
}

func NewPatternClassifierAdapter(classifier *core.PatternClassifier) *PatternClassifierAdapter {
	return &PatternClassifierAdapter{
		classifier: classifier,
	}
}

func (p *PatternClassifierAdapter) Classify(app, title, url string) (string, bool) {
	return p.classifier.Classify(app, title, url)
}
