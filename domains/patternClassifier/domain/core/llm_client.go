package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

type LLMClient struct {
	client  *openai.Client
	timeout time.Duration
}

func NewLLMClient() *LLMClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil // Return nil if no API key is set
	}

	return &LLMClient{
		client:  openai.NewClient(apiKey),
		timeout: 30 * time.Second,
	}
}

func (l *LLMClient) ClassifyUsage(app, title, url string) (string, error) {
	if l == nil || l.client == nil {
		return "", fmt.Errorf("LLM client not initialized")
	}

	prompt := l.buildPrompt(app, title, url)
	systemPrompt := l.buildSystemPrompt()

	ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
	defer cancel()

	resp, err := l.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       openai.GPT4Dot1,
		Temperature: 0.1,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})

	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	category := strings.TrimSpace(resp.Choices[0].Message.Content)

	return l.validateCategory(category), nil
}

func (l *LLMClient) buildPrompt(app, title, url string) string {
	var parts []string

	if app != "" {
		parts = append(parts, fmt.Sprintf("Application: %s", app))
	}
	if title != "" {
		parts = append(parts, fmt.Sprintf("Title: %s", title))
	}
	if url != "" {
		parts = append(parts, fmt.Sprintf("URL: %s", url))
	}

	return strings.Join(parts, "\n")
}

func (l *LLMClient) buildSystemPrompt() string {
	return "You are a usage categorization expert. Based on the user's active application usage pattern, categorize their current behavior into one of the predefined categories.\n\n**Analysis Context:**\n- App Name: The specific application the user is currently using\n- Title: The window title or content description\n- URL: The web address or application context (if applicable)\n\n**Instructions:**\n1. Analyze the user's digital behavior pattern from the provided app usage data\n2. Consider the app's primary function and the specific context (title/URL)\n3. Infer the user's intent and activity type\n4. If user use youtube but title is not about entertainment, should categorize properly\n4. Respond with **exactly one** category from the list below\n5. **Do not provide explanations or additional text**\n\n**Categories:**\nSNS, Documentation, Design, Communication, LLM, Development, Productivity, Video Editing, Entertainment, File Management, System & Utilities, Game, Education, Finance, Browsing, Marketing, Music, E-commerce & Shopping"
}

func (l *LLMClient) validateCategory(category string) string {
	validCategories := map[string]bool{
		"SNS":                   true,
		"Documentation":         true,
		"Design":                true,
		"Communication":         true,
		"LLM":                   true,
		"Development":           true,
		"Productivity":          true,
		"Video Editing":         true,
		"Entertainment":         true,
		"File Management":       true,
		"System & Utilities":    true,
		"Game":                  true,
		"Education":             true,
		"Finance":               true,
		"Browsing":              true,
		"Marketing":             true,
		"E-commerce & Shopping": true,
	}

	if validCategories[category] {
		return category
	}
	return "Uncategorized"
}
