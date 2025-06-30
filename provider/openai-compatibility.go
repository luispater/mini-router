package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/luispater/mini-router/config"
	_const "github.com/luispater/mini-router/const"
	"github.com/luispater/mini-router/models"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// / NewProviderOpenAICompatibility creates a new OpenAICompatibility provider.
func NewProviderOpenAICompatibility(_ *config.Config) (Provider, error) {
	// Define the supported models and their features.
	return &OpenAICompatibility{}, nil
}

// / init registers the provider.
func init() {
	RegisterProvider(_const.ProviderOpenAICompatibility, NewProviderOpenAICompatibility)
}

// apiKeyCounters stores the API key counters for each model
var apiKeyCounters sync.Map

// / OpenAICompatibility implements the Provider interface for OpenAI models.
type OpenAICompatibility struct {
	// baseUrl is the base URL for the API.
	baseUrl string
	// baseUrlDirect indicates whether to use the baseUrl directly.
	baseUrlDirect bool
}

// getAPIKey selects an API key from the model's API key list
func (p *OpenAICompatibility) getAPIKey(model models.Model) string {
	if len(model.ProviderAPIKey) == 0 {
		return ""
	}

	// Get the model's counter
	counterValue, _ := apiKeyCounters.LoadOrStore(model.ID, int64(0))
	counter := counterValue.(int64)

	// Calculate the index and get the API key
	index := int(counter) % len(model.ProviderAPIKey)
	apiKey := model.ProviderAPIKey[index]

	// Increment the counter
	apiKeyCounters.Store(model.ID, counter+1)

	// log.Printf("Using API key: %s", apiKey)

	return apiKey
}

// / SetBaseUrl sets the base URL for the API.
func (p *OpenAICompatibility) SetBaseUrl(url string, direct ...bool) {
	// Set the base URL.
	p.baseUrl = url
	// If the direct parameter is provided, set baseUrlDirect.
	if len(direct) > 0 {
		p.baseUrlDirect = direct[0]
	}
}

// / GetProviderType returns the provider's type.
func (p *OpenAICompatibility) GetProviderType() _const.ProviderType {
	return _const.ProviderOpenAICompatibility
}

// / CreateChatCompletion creates a chat completion.
func (p *OpenAICompatibility) CreateChatCompletion(ctx context.Context, cancel context.CancelFunc, request []byte, model models.Model, usage *Usage) ([]byte, error, []byte) {
	// If the model only supports streaming, use streaming.
	if model.StreamOnly {
		return p.CreateChatCompletionUseStream(ctx, cancel, request, model, usage)
	}

	// Define the URL.
	var url string
	// If baseUrlDirect is true, use the baseUrl directly.
	if p.baseUrlDirect {
		url = p.baseUrl
	} else {
		// Otherwise, build the URL for chat completion.
		url = fmt.Sprintf("%s/chat/completions", p.baseUrl)
	}

	// Create an HTTP request.
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(request))
	// If creating the request fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Set the request headers.
	req.Header.Set("Content-Type", "application/json")
	// Get the API key and set the Authorization header.
	apiKey := p.getAPIKey(model)
	if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}

	// Use http.Client to send the request.
	client := p.newHttpClient()
	resp, err := client.Do(req)
	// If sending the request fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Defer closing the response body.
	defer func() {
		err = resp.Body.Close()
	}()

	// Read the response body.
	data, err := io.ReadAll(resp.Body)
	// If reading the response body fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Check the response status code.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status), data
	}

	// Check if the response contains the id and object fields.
	responseIdResult := gjson.GetBytes(data, "id")
	responseObjectResult := gjson.GetBytes(data, "object")
	// If the response does not contain the id and object fields, return an error.
	if responseIdResult.Type == gjson.Null && responseObjectResult.Type == gjson.Null {
		return nil, fmt.Errorf("unexpected response: %s", string(data)), data
	}

	// Delete the provider field.
	data, _ = sjson.DeleteBytes(data, "provider")
	// Set the model field.
	data, _ = sjson.SetBytes(data, "model", model.Name)

	// If the response contains usage information, parse the usage information.
	if bytes.Contains(data, _const.TagPromptTokens) {
		// Parse prompt_tokens.
		promptTokensResult := gjson.GetBytes(data, "usage.prompt_tokens")
		if promptTokensResult.Type == gjson.Number {
			usage.PromptTokens = int(promptTokensResult.Int())
		}

		// Parse completion_tokens.
		completionTokensResult := gjson.GetBytes(data, "usage.completion_tokens")
		if completionTokensResult.Type == gjson.Number {
			usage.CompletionTokens = int(completionTokensResult.Int())
		}

		// Parse total_tokens.
		totalTokensResult := gjson.GetBytes(data, "usage.total_tokens")
		if totalTokensResult.Type == gjson.Number {
			usage.TotalTokens = int(totalTokensResult.Int())
		}

		// Parse cached_tokens.
		cachedTokensResult := gjson.GetBytes(data, "usage.prompt_tokens_details.cached_tokens")
		if cachedTokensResult.Type == gjson.Number {
			usage.PromptTokensDetails.CachedTokens = int(cachedTokensResult.Int())
		}

		// Parse audio_tokens.
		promptAudioTokensResult := gjson.GetBytes(data, "usage.prompt_tokens_details.audio_tokens")
		if promptAudioTokensResult.Type == gjson.Number {
			usage.PromptTokensDetails.AudioTokens = int(promptAudioTokensResult.Int())
		}

		// Parse audio_tokens.
		completionAudioTokensResult := gjson.GetBytes(data, "usage.completion_tokens_details.audio_tokens")
		if completionAudioTokensResult.Type == gjson.Number {
			usage.CompletionTokensDetails.AudioTokens = int(completionAudioTokensResult.Int())
		}

		// Parse reasoning_tokens.
		completionReasoningTokensResult := gjson.GetBytes(data, "usage.completion_tokens_details.reasoning_tokens")
		if completionReasoningTokensResult.Type == gjson.Number {
			usage.CompletionTokensDetails.ReasoningTokens = int(completionReasoningTokensResult.Int())
		}
	}

	return data, err, nil
}

// / CreateChatCompletionStream creates a streaming chat completion.
func (p *OpenAICompatibility) CreateChatCompletionStream(ctx context.Context, cancel context.CancelFunc, request []byte, model models.Model, usage *Usage) (io.ReadCloser, error, []byte) {
	// Define the URL.
	var url string
	// If baseUrlDirect is true, use the baseUrl directly.
	if p.baseUrlDirect {
		url = p.baseUrl
	} else {
		// Otherwise, build the chat completion URL.
		url = fmt.Sprintf("%s/chat/completions", p.baseUrl)
	}

	// Set stream_options.include_usage to true.
	request, err := sjson.SetBytes(request, "stream_options.include_usage", true)
	// If setting fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Create an HTTP request.
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(request))
	// If creating the request fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Set the request headers.
	req.Header.Set("Content-Type", "application/json")
	// Get the API key and set the Authorization header.
	apiKey := p.getAPIKey(model)
	if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
	// Set the Accept header to text/event-stream.
	req.Header.Set("Accept", "text/event-stream")

	// Use http.Client to send the request.
	client := p.newHttpClient()
	resp, err := client.Do(req)
	// If sending the request fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Check the response status code.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status), body
	}

	// Create a pipe.
	pr, pw := io.Pipe()
	// Create a buffered reader.
	reader := bufio.NewReader(resp.Body)

	// Start a goroutine to handle the streaming response.
	go func() {
		// Defer closing the pipe and the response body.
		defer func() {
			err = pw.Close()
			err = resp.Body.Close()
		}()

		// Loop to read the streaming response.
		for {
			// Read a line of data.
			var line []byte
			line, err = reader.ReadBytes('\n')
			// If reading fails, handle the error.
			if err != nil {
				// If it is EOF, it means reading is complete.
				if err == io.EOF {
					// If there is still data, write it to the pipe.
					if len(line) > 0 {
						_, err = pw.Write(bytes.TrimSpace(line))
					}
					break
				}
				// If it is another error, cancel the request.
				cancel()
				// If it is another error, close the pipe and return the error.
				// _ = pw.CloseWithError(fmt.Errorf("error reading stream: %w", err))
				// _ = pr.CloseWithError(fmt.Errorf("stop writing stream: %w", err))
				return
			}

			// Trim leading and trailing spaces from the line.
			line = bytes.TrimSpace(line)
			// If it is an empty line, skip it.
			if len(line) == 0 {
				continue
			}

			// If it starts with data:, process the data.
			if bytes.HasPrefix(line, _const.TagData) {
				// Remove the data: prefix.
				data := bytes.TrimPrefix(line, _const.TagData)
				// If it is [DONE], it means the streaming response has ended.
				if bytes.Equal(data, _const.TagDataDone) {
					_, err = pw.Write([]byte("data: [DONE]\n\n"))
					err = pw.Close()
					break
				}

				// Delete the provider field.
				data, _ = sjson.DeleteBytes(data, "provider")
				// Set the model field.
				data, _ = sjson.SetBytes(data, "model", model.Name)

				// If the response contains usage information, parse the usage information.
				if bytes.Contains(data, _const.TagPromptTokens) {
					// Parse prompt_tokens.
					promptTokensResult := gjson.GetBytes(data, "usage.prompt_tokens")
					if promptTokensResult.Type == gjson.Number {
						usage.PromptTokens = int(promptTokensResult.Int())
					}

					// Parse completion_tokens.
					completionTokensResult := gjson.GetBytes(data, "usage.completion_tokens")
					if completionTokensResult.Type == gjson.Number {
						usage.CompletionTokens = int(completionTokensResult.Int())
					}

					// Parse total_tokens.
					totalTokensResult := gjson.GetBytes(data, "usage.total_tokens")
					if totalTokensResult.Type == gjson.Number {
						usage.TotalTokens = int(totalTokensResult.Int())
					}

					// Parse cached_tokens.
					cachedTokensResult := gjson.GetBytes(data, "usage.prompt_tokens_details.cached_tokens")
					if cachedTokensResult.Type == gjson.Number {
						usage.PromptTokensDetails.CachedTokens = int(cachedTokensResult.Int())
					}

					// Parse audio_tokens.
					promptAudioTokensResult := gjson.GetBytes(data, "usage.prompt_tokens_details.audio_tokens")
					if promptAudioTokensResult.Type == gjson.Number {
						usage.PromptTokensDetails.AudioTokens = int(promptAudioTokensResult.Int())
					}

					// Parse audio_tokens.
					completionAudioTokensResult := gjson.GetBytes(data, "usage.completion_tokens_details.audio_tokens")
					if completionAudioTokensResult.Type == gjson.Number {
						usage.CompletionTokensDetails.AudioTokens = int(completionAudioTokensResult.Int())
					}

					// Parse reasoning_tokens
					completionReasoningTokensResult := gjson.GetBytes(data, "usage.completion_tokens_details.reasoning_tokens")
					if completionReasoningTokensResult.Type == gjson.Number {
						usage.CompletionTokensDetails.ReasoningTokens = int(completionReasoningTokensResult.Int())
					}
				}

				// Construct the output data.
				output := []byte("data: ")
				output = append(output, data...)
				output = append(output, []byte("\n\n")...)
				// Write the data to the pipe.
				_, err = pw.Write(output)
			}
		}
	}()

	return pr, nil, nil
}

// / CreateChatCompletionUseStream creates a chat completion using streaming.
func (p *OpenAICompatibility) CreateChatCompletionUseStream(ctx context.Context, cancel context.CancelFunc, request []byte, model models.Model, usage *Usage) ([]byte, error, []byte) {
	// Set stream to true to use the streaming API internally.
	streamRequest, err := json.Marshal(map[string]interface{}{
		"stream": true,
	})
	// If serialization fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Merge the original request with the stream flag.
	request, err = jsonMerge(request, streamRequest)
	// If merging fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	// Call the streaming API.
	stream, err, errBody := p.CreateChatCompletionStream(ctx, cancel, request, model, usage)
	// If the call fails, return an error.
	if err != nil {
		return nil, err, errBody
	}
	// Defer closing the stream.
	defer func() {
		err = stream.Close()
	}()

	// Read all chunks from the stream.
	reader := bufio.NewReader(stream)
	// Define the full response.
	var fullResponse *ChatCompletionResponse
	// Define the content.
	var content string
	// Define the tool calls.
	var toolCalls []map[string]interface{}

	// Loop to read the stream.
	for {
		// Read a line of data.
		var line []byte
		line, err = reader.ReadBytes('\n')
		// If reading fails, handle the error.
		if err != nil {
			// If it is EOF, it means reading is complete.
			if err == io.EOF {
				break
			}
			return nil, err, nil
		}

		// Trim leading and trailing spaces from the line.
		line = bytes.TrimSpace(line)
		// If it is an empty line, skip it.
		if len(line) == 0 {
			continue
		}

		// Check if it is a data line.
		if bytes.HasPrefix(line, []byte("data: ")) {
			// Remove the data: prefix.
			data := bytes.TrimPrefix(line, []byte("data: "))

			// Check for the [DONE] marker.
			if bytes.Equal(data, []byte("[DONE]")) {
				break
			}

			// Parse the chunk.
			var chunk map[string]interface{}
			if err = json.Unmarshal(data, &chunk); err != nil {
				continue
			}

			// If not yet initialized, initialize the full response.
			if fullResponse == nil {
				fullResponse = &ChatCompletionResponse{
					ID:                chunk["id"].(string),
					Object:            "chat.completion",
					Created:           int64(chunk["created"].(float64)),
					Model:             chunk["model"].(string),
					SystemFingerprint: "",
					Choices:           []ChatCompletionChoice{},
					Usage:             Usage{},
				}

				// Check if system_fingerprint exists.
				if fingerprint, ok := chunk["system_fingerprint"]; ok {
					fullResponse.SystemFingerprint = fingerprint.(string)
				}
			}

			// Extract content and tool calls from the chunk.
			if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
				choice := choices[0].(map[string]interface{})

				// If delta exists, process the delta.
				if delta, deltaOK := choice["delta"].(map[string]interface{}); deltaOK {
					// Append content.
					if deltaContent, deltaContentOK := delta["content"].(string); deltaContentOK && deltaContent != "" {
						content += deltaContent
					}

					// Process tool calls.
					if deltaToolCalls, deltaToolCallsOK := delta["tool_calls"].([]interface{}); deltaToolCallsOK && len(deltaToolCalls) > 0 {
						// Iterate over tool calls.
						for _, tc := range deltaToolCalls {
							toolCall := tc.(map[string]interface{})

							// Find an existing tool call with the same index or create a new one.
							var existingToolCall map[string]interface{}
							toolCallIndex := int(toolCall["index"].(float64))

							// Ensure toolCalls has enough length.
							for len(toolCalls) <= toolCallIndex {
								toolCalls = append(toolCalls, map[string]interface{}{
									"index": len(toolCalls),
									"id":    "",
									"type":  "",
									"function": map[string]interface{}{
										"name":      "",
										"arguments": "",
									},
								})
							}

							existingToolCall = toolCalls[toolCallIndex]

							// Update tool call properties.
							if id, idOK := toolCall["id"].(string); idOK && id != "" {
								existingToolCall["id"] = id
							}

							if tcType, tcTypeOK := toolCall["type"].(string); tcTypeOK && tcType != "" {
								existingToolCall["type"] = tcType
							}

							if function, functionOK := toolCall["function"].(map[string]interface{}); functionOK {
								existingFunc := existingToolCall["function"].(map[string]interface{})

								if name, nameOK := function["name"].(string); nameOK && name != "" {
									existingFunc["name"] = name
								}

								if args, argsOK := function["arguments"].(string); argsOK {
									existingArgs := existingFunc["arguments"].(string)
									existingFunc["arguments"] = existingArgs + args
								}
							}
						}
					}

					// If available, get the completion reason.
					if finishReason, finishReasonOK := choice["finish_reason"].(string); finishReasonOK && finishReason != "" {
						if len(fullResponse.Choices) == 0 {
							fullResponse.Choices = append(fullResponse.Choices, ChatCompletionChoice{
								Index:        0,
								FinishReason: finishReason,
							})
						} else {
							fullResponse.Choices[0].FinishReason = finishReason
						}
					}
				}
			}

			// If available, extract usage information.
			if usageData, ok := chunk["usage"].(map[string]interface{}); ok {
				if promptTokens, promptTokensOK := usageData["prompt_tokens"].(float64); promptTokensOK {
					usage.PromptTokens = int(promptTokens)
				}

				if completionTokens, completionTokensOK := usageData["completion_tokens"].(float64); completionTokensOK {
					usage.CompletionTokens = int(completionTokens)
				}

				if totalTokens, totalTokensOK := usageData["total_tokens"].(float64); totalTokensOK {
					usage.TotalTokens = int(totalTokens)
				}

				// Process prompt tokens details.
				if promptTokensDetails, promptTokensDetailsOK := usageData["prompt_tokens_details"].(map[string]interface{}); promptTokensDetailsOK {
					if cachedTokens, cachedTokensOK := promptTokensDetails["cached_tokens"].(float64); cachedTokensOK {
						usage.PromptTokensDetails.CachedTokens = int(cachedTokens)
					}

					if audioTokens, audioTokensOK := promptTokensDetails["audio_tokens"].(float64); audioTokensOK {
						usage.PromptTokensDetails.AudioTokens = int(audioTokens)
					}
				}

				// Process completion tokens details.
				if completionTokensDetails, completionTokensDetailsOK := usageData["completion_tokens_details"].(map[string]interface{}); completionTokensDetailsOK {
					if audioTokens, audioTokensOK := completionTokensDetails["audio_tokens"].(float64); audioTokensOK {
						usage.CompletionTokensDetails.AudioTokens = int(audioTokens)
					}

					if reasoningTokens, reasoningTokensOK := completionTokensDetails["reasoning_tokens"].(float64); reasoningTokensOK {
						usage.CompletionTokensDetails.ReasoningTokens = int(reasoningTokens)
					}
				}
			}
		}
	}

	// If we did not get a valid response, return an error.
	if fullResponse == nil {
		return nil, errors.New("failed to parse streaming response"), nil
	}

	// Create the final message with content and tool calls.
	message := Message{
		Role:    "assistant",
		Content: content,
	}

	// If there are tool calls, add them.
	if len(toolCalls) > 0 {
		// Convert tool calls to the format expected in the response.
		messageToolCalls := make([]map[string]interface{}, len(toolCalls))
		for i, tc := range toolCalls {
			messageToolCalls[i] = tc
		}

		// Add tool calls to the message metadata.
		message.Meta = map[string]interface{}{
			"tool_calls": messageToolCalls,
		}
	}

	// Set the message in the response.
	if len(fullResponse.Choices) == 0 {
		fullResponse.Choices = append(fullResponse.Choices, ChatCompletionChoice{
			Index:   0,
			Message: message,
		})
	} else {
		fullResponse.Choices[0].Message = message
	}

	// Set the usage information.
	fullResponse.Usage = *usage

	// Convert the response to JSON.
	responseBytes, err := json.Marshal(fullResponse)
	// If conversion fails, return an error.
	if err != nil {
		return nil, err, nil
	}

	return responseBytes, nil, nil
}

// / Close closes the provider.
func (p *OpenAICompatibility) Close() error {
	return nil
}

// / newHttpClient creates a new HTTP client.
func (p *OpenAICompatibility) newHttpClient() *http.Client {
	// Create an HTTP transport.
	transport := &http.Transport{
		// Set the connection context.
		DialContext: (&net.Dialer{
			// Set the connection timeout to 3 seconds.
			Timeout: 3 * time.Second,
			// Set keep-alive.
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	// Create an HTTP client without a global timeout.
	return &http.Client{
		// Set the transport.
		Transport: transport,
	}
}
