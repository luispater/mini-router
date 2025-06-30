package provider

import (
	"context"
	"encoding/json"
	"github.com/luispater/mini-router/config"
	_const "github.com/luispater/mini-router/const"
	"github.com/luispater/mini-router/models"
	"github.com/tidwall/sjson"
	"io"
)

// Message represents a chat message
type Message struct {
	// Role is the role of the message sender
	Role string `json:"role"`
	// Content is the content of the message
	Content string `json:"content"`
	// Images is the image data for multimodal models
	Images [][]byte `json:"-"`
	// Name is the name of the message sender
	Name string `json:"name,omitempty"`
	// Meta is the metadata
	Meta map[string]interface{} `json:"-"`
}

// MessageArrayContentItemImageURL represents the image URL in a message array content item
type MessageArrayContentItemImageURL struct {
	// URL is the image URL
	URL string `json:"url,omitempty"`
}

// MessageArrayContentItem represents a content item in a message array
type MessageArrayContentItem struct {
	// Type is the type of the content item
	Type string `json:"type"`
	// Text is the text content
	Text string `json:"text,omitempty"`
	// ImageURL is the image URL
	ImageURL MessageArrayContentItemImageURL `json:"image_url,omitempty"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	// ID is the response ID
	ID string `json:"id"`
	// Object is the object type
	Object string `json:"object"`
	// Created is the creation time
	Created int64 `json:"created"`
	// Model is the model used
	Model string `json:"model"`
	// SystemFingerprint is the system fingerprint
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
	// Choices is the list of choices
	Choices []ChatCompletionChoice `json:"choices"`
	// Usage is the usage information
	Usage Usage `json:"usage"`
}

// ChatCompletionChoice represents a choice in a chat completion response
type ChatCompletionChoice struct {
	// Index is the choice index
	Index int `json:"index"`
	// Message is the message content
	Message Message `json:"message"`
	// FinishReason is the reason for finishing
	FinishReason string `json:"finish_reason"`
	// LogProbs is the log probabilities
	LogProbs map[string]interface{} `json:"logprobs,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	// PromptTokens is the number of prompt tokens
	PromptTokens int `json:"prompt_tokens"`
	// CompletionTokens is the number of completion tokens
	CompletionTokens int `json:"completion_tokens"`
	// TotalTokens is the total number of tokens
	TotalTokens int `json:"total_tokens"`
	// PromptTokensDetails is the detailed prompt token usage information
	PromptTokensDetails UsagePromptTokensDetails `json:"prompt_tokens_details"`
	// CompletionTokensDetails is the detailed completion token usage information
	CompletionTokensDetails UsageCompletionTokensDetails `json:"completion_tokens_details"`
}

// UsagePromptTokensDetails represents detailed prompt token usage information
type UsagePromptTokensDetails struct {
	// CachedTokens is the number of cached tokens
	CachedTokens int `json:"cached_tokens"`
	// AudioTokens is the number of audio tokens
	AudioTokens int `json:"audio_tokens"`
}

// UsageCompletionTokensDetails represents detailed completion token usage information
type UsageCompletionTokensDetails struct {
	// AudioTokens is the number of audio tokens
	AudioTokens int `json:"audio_tokens"`
	// ReasoningTokens is the number of reasoning tokens
	ReasoningTokens int `json:"reasoning_tokens"`
}

// Provider defines the interface that all AI providers must implement
type Provider interface {
	// GetProviderType returns the type of the provider
	GetProviderType() _const.ProviderType

	// CreateChatCompletion creates a chat completion
	CreateChatCompletion(ctx context.Context, cancel context.CancelFunc, request []byte, model models.Model, usage *Usage) ([]byte, error, []byte)

	// CreateChatCompletionStream creates a streaming chat completion
	CreateChatCompletionStream(ctx context.Context, cancel context.CancelFunc, request []byte, model models.Model, usage *Usage) (io.ReadCloser, error, []byte)

	// Close closes any resources used by the provider
	Close() error
}

// ProviderFactory is a function that creates a new provider instance
type ProviderFactory func(cfg *config.Config) (Provider, error)

// ProviderRegistry stores all available providers
var ProviderRegistry = make(map[_const.ProviderType]ProviderFactory)

// RegisterProvider registers a new provider factory
func RegisterProvider(providerType _const.ProviderType, factory ProviderFactory) {
	ProviderRegistry[providerType] = factory
}

// jsonMerge merges two JSON objects
func jsonMerge(original, patch []byte) ([]byte, error) {
	// Parse the patch JSON
	var patchObj map[string]interface{}
	if err := json.Unmarshal(patch, &patchObj); err != nil {
		return nil, err
	}

	// Apply each field from the patch to the original
	result := original
	var err error
	for key, value := range patchObj {
		result, err = sjson.SetBytes(result, key, value)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
