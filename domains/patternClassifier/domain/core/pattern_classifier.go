package core

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"pomocore-data/domains/patternClassifier/domain/structure"
	"pomocore-data/infrastructure/mongoDB/model"
	"strings"
	"sync"
)

type PatternClassifier struct {
	appTrie         *structure.Trie
	urlTrie         *structure.AhoCorasick
	cache           *sync.Map
	llmClient       *LLMClient
	initialized     bool
	categoryToIdMap map[string]primitive.ObjectID
}

func NewPatternClassifier() *PatternClassifier {
	return &PatternClassifier{
		cache:       &sync.Map{},
		llmClient:   NewLLMClient(),
		initialized: false,
	}
}

func (p *PatternClassifier) Initialize(patterns []model.CategoryPattern) {
	p.appTrie = p.initAppTrie(patterns)
	p.urlTrie = p.initUrlAhoCorasick(patterns)
	p.initialized = true
}

func (p *PatternClassifier) initAppTrie(patterns []model.CategoryPattern) *structure.Trie {
	trie := structure.NewTrie()
	for _, pattern := range patterns {
		for _, app := range pattern.AppPatterns {
			trie.Insert(app, pattern.Category)
		}
	}
	return trie
}

func (p *PatternClassifier) initUrlAhoCorasick(patterns []model.CategoryPattern) *structure.AhoCorasick {
	ac := structure.NewAhoCorasick()
	for _, pattern := range patterns {
		for _, domain := range pattern.DomainPatterns {
			ac.Insert(domain, pattern.Category)
		}
	}
	ac.Connect()
	return ac
}

func (p *PatternClassifier) Classify(app, title, url string) (string, bool) {
	if !p.initialized {
		log.Fatal("PatternClassifier not initialized")
	}
	app = strings.ToLower(app)

	//TODO: 소문자 처리
	var category string

	if category = p.classifyFromApp(app); category != "" {
		return category, false
	}

	if category = p.classifyFromURL(url); category != "" {
		return category, false
	}

	query := getQuery(app, title, url)
	if category = p.classifyFromCache(query); category != "" {
		return category, true
	}

	if category = p.classifyFromLLM(app, title, url); category != "" {
		return p.putCache(query, category), true
	}

	return "", true
}

func (p *PatternClassifier) ClassifyFromApp(app string) string {
	if p.appTrie == nil {
		return ""
	}
	return p.appTrie.Search(app)
}

func (p *PatternClassifier) ClassifyFromURL(url string) string {
	if p.urlTrie == nil {
		return ""
	}
	return p.urlTrie.Search(url)
}

func (p *PatternClassifier) classifyFromApp(app string) string {
	return p.ClassifyFromApp(app)
}

func (p *PatternClassifier) classifyFromURL(url string) string {
	return p.ClassifyFromURL(url)
}

func (p *PatternClassifier) classifyFromCache(query string) string {
	if value, exists := p.cache.Load(query); exists {
		return value.(string)
	}
	return ""
}

func (p *PatternClassifier) classifyFromLLM(app, title, url string) string {
	if p.llmClient == nil {
		log.Printf("LLM client is nil - OPENAI_API_KEY not set?")
		return ""
	}

	log.Printf("Calling LLM for classification: app=%s, title=%s, url=%s", app, title, url)
	category, err := p.llmClient.ClassifyUsage(app, title, url)
	if err != nil {
		log.Printf("LLM classification failed: %v", err)
		return ""
	}

	log.Printf("LLM returned category: %s", category)
	return category
}

func (p *PatternClassifier) putCache(query, category string) string {
	p.cache.Store(query, category)
	return category
}

func getQuery(app, title, url string) string {
	return fmt.Sprintf("app: %s, title: %s, url: %s", app, title, url)
}
