package models

import (
	"time"
)

// Model represents an AI model from a provider
type Model struct {
	// ID is the primary key
	ID uint `json:"id" yaml:"id"`
	// CreatedAt is the creation time
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	// UpdatedAt is the update time
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`

	// Name is the model name
	Name string `json:"name" yaml:"name"`
	// ProviderModelName is the provider model name
	ProviderModelName string `json:"provider_model_name" yaml:"provider_model_name"`
	// DisplayName is the display name
	DisplayName string `json:"display_name" yaml:"display_name"`
	// Description is the description
	Description string `json:"description" yaml:"description"`

	// Model capabilities
	// SupportsChat indicates whether chat is supported
	SupportsChat bool `json:"supports_chat" yaml:"supports_chat"`
	// SupportsCompletion indicates whether completion is supported
	SupportsCompletion bool `json:"supports_completion" yaml:"supports_completion"`
	// SupportsEmbedding indicates whether embedding is supported
	SupportsEmbedding bool `json:"supports_embedding" yaml:"supports_embedding"`
	// SupportsInputImage indicates whether image input is supported
	SupportsInputImage bool `json:"supports_input_image" yaml:"supports_input_image"`
	// SupportsImageGen indicates whether image generation is supported
	SupportsImageGen bool `json:"supports_image_gen" yaml:"supports_image_gen"`
	// SupportsImageEdit indicates whether image editing is supported
	SupportsImageEdit bool `json:"supports_image_edit" yaml:"supports_image_edit"`
	// SupportsImageVar indicates whether image variation is supported
	SupportsImageVar bool `json:"supports_image_var" yaml:"supports_image_var"`
	// SupportsAudioTrans indicates whether audio transcription is supported
	SupportsAudioTrans bool `json:"supports_audio_trans" yaml:"supports_audio_trans"`
	// SupportsAudioTrans2 indicates whether audio translation is supported
	SupportsAudioTrans2 bool `json:"supports_audio_trans2" yaml:"supports_audio_trans2"`

	SupportGoogleThinking bool `json:"support_google_thinking" yaml:"support_google_thinking"`

	// Rate limiting
	// RPM is the requests per minute
	RPM int `json:"rpm" yaml:"rpm"` // Requests per minute
	// RPH is the requests per hour
	RPH int `json:"rph" yaml:"rph"` // Requests per hour
	// RPD is the requests per day
	RPD int `json:"rpd" yaml:"rpd"` // Requests per day
	// TPM is the tokens per minute
	TPM int `json:"tpm" yaml:"tpm"` // Tokens per minute
	// TPH is the tokens per hour
	TPH int `json:"tph" yaml:"tph"` // Tokens per hour
	// TPD is the tokens per day
	TPD int `json:"tpd" yaml:"tpd"` // Tokens per day

	// MaxTokens is the maximum number of output tokens
	MaxTokens int `json:"max_tokens" yaml:"max_tokens"` // Max output tokens
	// ContextLength is the maximum context length
	ContextLength       int      `json:"context_length" yaml:"context_length"` // Max context length
	SupportedParameters []string `json:"supported_parameters" yaml:"supported_parameters"`

	// Price (in USD)
	// InputPricePerToken is the price per input token (in USD)
	InputPricePerToken float64 `json:"input_price_per_token" yaml:"input_price_per_token"`
	// OutputPricePerToken is the price per output token (in USD)
	OutputPricePerToken float64 `json:"output_price_per_token" yaml:"output_price_per_token"`

	// Relationships
	// BaseURL is the base URL
	BaseURL string `json:"base_url" yaml:"base_url"`
	// BaseURLDirect indicates whether to use the base URL directly
	BaseURLDirect bool `json:"base_url_direct" yaml:"base_url_direct"`
	// IsOpenAICompatibility indicates whether it is compatible with the OpenAI format
	IsOpenAICompatibility bool `json:"is_openai_compatibility" yaml:"is_openai_compatibility"`
	// StreamOnly indicates whether only streaming is supported
	StreamOnly bool `json:"stream_only" yaml:"stream_only"`

	// ProviderAPIKey is the list of provider API keys for this model
	ProviderAPIKey []string `json:"provider_api_key" yaml:"provider_api_key"`
	// Enabled indicates whether the model is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Visible indicates whether the model is visible
	Visible bool `json:"visible" yaml:"visible"`
}

type DisplayModel struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Object              string   `json:"object"`
	Description         string   `json:"description"`
	SupportedParameters []string `json:"supported_parameters"`
	ContextLength       int      `json:"context_length,omitempty"`
	MaxCompletionTokens int      `json:"max_completion_tokens,omitempty"`
	Architecture        struct {
		Modality         string   `json:"modality"`
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture"`
	Pricing struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing,omitempty"`
}
