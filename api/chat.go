package api

import (
	"bytes"
	"context"
	"fmt"
	jsonschema "github.com/luispater/mini-router/json-schema"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/luispater/mini-router/config"
	_const "github.com/luispater/mini-router/const"
	"github.com/tidwall/sjson"

	"github.com/gin-gonic/gin"
	"github.com/luispater/mini-router/models"
	"github.com/luispater/mini-router/provider"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"
)

// lastUsedModelIndex tracks the last used model index for each model name to implement round-robin load balancing
var (
	// lastUsedModelIndex stores the last used index for each model name
	lastUsedModelIndex = make(map[string]int)
	// modelIndexMutexes stores the mutex for each model name
	modelIndexMutexes = make(map[string]*sync.Mutex)
	// mutexMapLock is used to protect concurrent access to modelIndexMutexes
	mutexMapLock = &sync.Mutex{}
)

// schemaLoader is used to load the JSON schema
var schemaLoader gojsonschema.JSONLoader

// init function is executed when the package is initialized
func init() {
	// Create a new byte-based JSON schema loader from the embedded file
	schemaLoader = gojsonschema.NewBytesLoader(jsonschema.ChatCompletionsSchema)
}

// validateRequest validates whether the request conforms to the JSON schema
func validateRequest(request []byte) (*gojsonschema.Result, error) {
	// Create a new byte-based document loader
	documentLoader := gojsonschema.NewBytesLoader(request)
	// Use the schema loader and document loader for validation
	return gojsonschema.Validate(schemaLoader, documentLoader)
}

// ChatCompletionHandler handles chat completion requests
func ChatCompletionHandler(cfg *config.Config, providerRegistry map[_const.ProviderType]provider.ProviderFactory) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set the response header, specifying the content type and character set
		c.Header("Content-Type", "application/json; charset=utf-8")

		// Get the raw JSON data
		rawJson, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err), "code": 400})
			return
		}

		customProviderNames := make([]string, 0)
		providerResult := gjson.GetBytes(rawJson, "provider")
		if providerResult.Type == gjson.String {
			customProviderNames = append(customProviderNames, providerResult.String())
			rawJson, _ = sjson.DeleteBytes(rawJson, "provider")
		} else if providerResult.Type == gjson.JSON {
			providerResult.ForEach(func(key, value gjson.Result) bool {
				customProviderNames = append(customProviderNames, strings.ToLower(value.String()))
				return true
			})
			rawJson, _ = sjson.DeleteBytes(rawJson, "provider")
		} else {
			providerHeader := c.GetHeader("Provider")
			if providerHeader != "" {
				splitProviderHeader := strings.Split(providerHeader, ",")
				for _, item := range splitProviderHeader {
					customProviderNames = append(customProviderNames, strings.ToLower(strings.TrimSpace(item)))
				}
			}
		}

		// Validate the request
		result, err := validateRequest(rawJson)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "code": 400})
			return
		}
		if !result.Valid() {
			for _, desc := range result.Errors() {
				errorMsg := fmt.Sprintf("Invalid request: %v", desc.Description())
				if desc.Type() == "invalid_type" || desc.Type() == "string_gte" {
					errorMsg = fmt.Sprintf("Invalid request: Field `%s` %s", desc.Field(), desc.Description())
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg, "code": 400})
				// log.Println(string(rawJson))
				return
			}
		}

		modelNameResult := gjson.GetBytes(rawJson, "model")
		if modelNameResult.Type != gjson.String {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: model field is missing or not a string", "code": 400})
			return
		}
		modelName := modelNameResult.String()

		// Filter models from memory
		var providerModels []models.Model
		allModels := cfg.Models

		if len(customProviderNames) > 0 {
			for _, m := range allModels {
				if m.Name == modelName && m.Enabled {
					providerModels = append(providerModels, m)
				}
			}
		} else {
			for _, m := range allModels {
				if m.Name == modelName && m.Enabled {
					providerModels = append(providerModels, m)
				}
			}
		}

		if len(providerModels) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": fmt.Sprintf("Model %s not found or not available.", modelName), "code": 404}})
			return
		}

		// Round-robin load balancing
		loadBalanceKey := modelName
		if len(customProviderNames) > 0 {
			sort.Strings(customProviderNames)
			loadBalanceKey = fmt.Sprintf("%s-%s", modelName, strings.Join(customProviderNames, "-"))
		}

		mutex, exists := modelIndexMutexes[loadBalanceKey]
		if !exists {
			mutexMapLock.Lock()
			mutex = &sync.Mutex{}
			modelIndexMutexes[loadBalanceKey] = mutex
			mutexMapLock.Unlock()
		}

		mutex.Lock()
		startIndex := lastUsedModelIndex[loadBalanceKey]
		lastUsedModelIndex[loadBalanceKey] = (startIndex + 1) % len(providerModels)
		mutex.Unlock()

		reorderedModels := make([]models.Model, len(providerModels))
		for i := 0; i < len(providerModels); i++ {
			reorderedModels[i] = providerModels[(startIndex+i)%len(providerModels)]
		}

		var finalErr error
		for _, model := range reorderedModels {
			factory, ok := providerRegistry[_const.ProviderOpenAICompatibility]

			if !ok {
				finalErr = fmt.Errorf("provider factory not found")
				log.Println(finalErr)
				continue
			}

			providerInstance, errFactory := factory(cfg)
			if errFactory != nil {
				finalErr = fmt.Errorf("failed to create provider: %v", errFactory)
				log.Println(finalErr)
				continue
			}

			if model.IsOpenAICompatibility {
				if instance, isOk := providerInstance.(*provider.OpenAICompatibility); isOk {
					instance.SetBaseUrl(model.BaseURL, model.BaseURLDirect)
				}
			}

			// If the model name contains gemini, check the reasoning_effort field
			if model.SupportGoogleThinking {
				reasoningEffortResult := gjson.GetBytes(rawJson, "reasoning_effort")
				if reasoningEffortResult.Type == gjson.String {
					if reasoningEffortResult.String() == "none" {
						rawJson, err = sjson.DeleteBytes(rawJson, "reasoning_effort")
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.thinking_budget", 0)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.include_thoughts", false)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
					} else if reasoningEffortResult.String() == "auto" {
						rawJson, err = sjson.DeleteBytes(rawJson, "reasoning_effort")
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.thinking_budget", -1)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.include_thoughts", true)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
					} else if reasoningEffortResult.String() == "low" {
						rawJson, err = sjson.DeleteBytes(rawJson, "reasoning_effort")
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.thinking_budget", 1024)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.include_thoughts", true)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
					} else if reasoningEffortResult.String() == "medium" {
						rawJson, err = sjson.DeleteBytes(rawJson, "reasoning_effort")
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.thinking_budget", 8192)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.include_thoughts", true)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
					} else if reasoningEffortResult.String() == "high" {
						rawJson, err = sjson.DeleteBytes(rawJson, "reasoning_effort")
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.thinking_budget", 24576)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
						rawJson, err = sjson.SetBytes(rawJson, "extra_body.google.thinking_config.include_thoughts", true)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
					} else {
						rawJson, err = sjson.DeleteBytes(rawJson, "reasoning_effort")
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "request body error", "code": 500})
							return
						}
					}
				}
			}

			isStream := gjson.GetBytes(rawJson, "stream").Bool()
			rawJson, _ = sjson.SetBytes(rawJson, "model", model.ProviderModelName)

			if isStream {
				finalErr = handleStreamingChatCompletion(c, providerInstance, rawJson, model)
			} else {
				rawJson, _ = sjson.DeleteBytes(rawJson, "stream_options")
				finalErr = handleNonStreamingChatCompletion(c, providerInstance, rawJson, model)
			}

			_ = providerInstance.Close()

			if finalErr == nil {
				// log.Printf("Request model %s OK\n", model.Name)
				return // Success, exit handler
			}
			log.Printf("Request model %s error: %s\n", model.ProviderModelName, finalErr.Error())
		}

		if finalErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"message": "All providers failed: " + finalErr.Error(), "code": 503}})
		}
	}
}

func handleStreamingChatCompletion(c *gin.Context, p provider.Provider, request []byte, model models.Model) error {
	// Set response headers for streaming
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	// Call the provider's streaming API
	usage := provider.Usage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
		PromptTokensDetails: provider.UsagePromptTokensDetails{
			CachedTokens: 0,
			AudioTokens:  0,
		},
		CompletionTokensDetails: provider.UsageCompletionTokensDetails{
			AudioTokens:     0,
			ReasoningTokens: 0,
		},
	}

	// Get the streaming response
	stream, err, errBody := p.CreateChatCompletionStream(ctx, cancel, request, model, &usage)
	// If getting the streaming response fails
	if err != nil {
		// If there is an error body
		if errBody != nil {
			// Write the error body to the response
			_, err = c.Writer.Write(errBody)
		}
		return err
	}
	// Defer closing the stream
	defer func() {
		err = stream.Close()
	}()

	// Use a channel to receive read results
	resultChan := make(chan []byte)
	resultEOFChan := make(chan bool)
	errChan := make(chan error)

	// Read data in a goroutine
	go func() {
		for {
			// Read data from the stream
			buffer := make([]byte, 4096)
			var n int
			n, err = stream.Read(buffer)
			// If EOF is reached
			if err == io.EOF {
				// Send an EOF signal
				resultEOFChan <- true
				return
			}
			// If an error occurs while reading
			if err != nil {
				// Send an error signal
				errChan <- err
				return
			}

			// Send the read data
			resultChan <- buffer[:n]
		}
	}()

	// Error return value
	var requestError error
	// Processing tag
	processingTag := []byte(": ROUTER-FOR-ME PROCESSING\n\n")
	// Stream the response to the client

	startGoogleThinking := false
	c.Stream(func(w io.Writer) bool {
		for {
			select {
			// If data is received
			case response := <-resultChan:
				// Trim leading and trailing whitespace
				tmp := bytes.TrimSpace(response)
				// If it does not start with : or data:
				if !bytes.HasPrefix(tmp, _const.TagNoData) && !bytes.HasPrefix(tmp, _const.TagData) {
					// Set the error message
					requestError = fmt.Errorf("unexpected response: %s", string(tmp))
					return false // stop c.Stream
				} else {
					if model.SupportGoogleThinking {
						response = bytes.TrimSpace(response[5:])
						thoughtResult := gjson.GetBytes(response, "choices.0.delta.extra_content.google.thought")
						if thoughtResult.Type == gjson.True {
							if !startGoogleThinking {
								startGoogleThinking = true
							}
							contentResult := gjson.GetBytes(response, "choices.0.delta.content")
							if contentResult.Type == gjson.String {
								reasoningContent := contentResult.String()
								if strings.HasPrefix(reasoningContent, "<thought>") {
									reasoningContent = reasoningContent[9:]
								}
								response, _ = sjson.SetBytes(response, "choices.0.delta.reasoning_content", reasoningContent)
								response, _ = sjson.DeleteBytes(response, "choices.0.delta.content")
								response, _ = sjson.DeleteBytes(response, "choices.0.delta.extra_content")
							}
						} else if startGoogleThinking {
							contentResult := gjson.GetBytes(response, "choices.0.delta.content")
							content := contentResult.String()
							if strings.HasPrefix(content, "</thought>") {
								content = content[10:]
								response, _ = sjson.SetBytes(response, "choices.0.delta.content", content)
								startGoogleThinking = false
							}
						}
						_, err = w.Write([]byte("data: "))
						_, err = w.Write(response)
						_, err = w.Write([]byte("\n\n"))
					} else {
						// Write the response
						_, err = w.Write(response)
					}
					return true // continue c.Stream
				}
			// If an error is received
			case err = <-errChan:
				requestError = err
				return false // stop c.Stream
			// If EOF is received
			case <-resultEOFChan:
				return false // stop c.Stream
			// If a timeout occurs
			case <-time.After(500 * time.Millisecond):
				// Write the processing tag
				_, err = w.Write(processingTag)
				return true // continue c.Stream
			}
		}
	})

	return requestError
}

func handleNonStreamingChatCompletion(c *gin.Context, p provider.Provider, request []byte, model models.Model) error {
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	// Call the provider's API
	usage := provider.Usage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
		PromptTokensDetails: provider.UsagePromptTokensDetails{
			CachedTokens: 0,
			AudioTokens:  0,
		},
		CompletionTokensDetails: provider.UsageCompletionTokensDetails{
			AudioTokens:     0,
			ReasoningTokens: 0,
		},
	}

	// Define the error response struct
	type ErrorResponse struct {
		Error error
		Body  []byte
	}

	var err error
	// Use a channel to receive read results
	resultChan := make(chan []byte)
	errChan := make(chan ErrorResponse)

	// Read data in a goroutine
	go func() {
		var response []byte
		var errBody []byte
		// Call the provider's CreateChatCompletion method
		response, err, errBody = p.CreateChatCompletion(ctx, cancel, request, model, &usage)
		// If there is an error
		if err != nil {
			// Send the error response
			errChan <- ErrorResponse{Error: err, Body: errBody}
			return
		}
		// Send the response
		resultChan <- response
	}()

	// Error return value
	var requestError error

	// Stream the response to the client
	c.Stream(func(w io.Writer) bool {
		for {
			select {
			// If data is received
			case response := <-resultChan:
				// Return the response
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.Writer.WriteHeader(http.StatusOK)
				response = bytes.TrimSpace(response)

				if model.SupportGoogleThinking {
					thoughtResult := gjson.GetBytes(response, "choices.0.message.extra_content.google.thought")
					if thoughtResult.Type == gjson.True {
						contentResult := gjson.GetBytes(response, "choices.0.message.content")
						if contentResult.Type == gjson.String {
							content := contentResult.String()
							thoughtStartIndex := strings.Index(content, "<thought>")
							thoughtEndIndex := strings.Index(content, "</thought>")
							if thoughtStartIndex != -1 && thoughtEndIndex != -1 {
								response, _ = sjson.SetBytes(response, "choices.0.message.reasoning_content", content[thoughtStartIndex+9:thoughtEndIndex])
								response, _ = sjson.SetBytes(response, "choices.0.message.content", content[:thoughtStartIndex]+content[thoughtEndIndex+10:])
								response, _ = sjson.DeleteBytes(response, "choices.0.message.extra_content")
							}
						}
					}
				}

				_, err = w.Write(response)
				return false // stop c.Stream
			// If an error is received
			case responseError := <-errChan:
				// c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create chat completion: %v", err)})
				// Set the error message
				requestError = responseError.Error
				// Write the error body
				_, err = c.Writer.Write(responseError.Body)
				return false // stop c.Stream
			// If a timeout occurs
			case <-time.After(500 * time.Millisecond):
				// Write a newline character
				_, err = w.Write([]byte{10})
				return true // continue c.Stream
			}
		}
	})

	return requestError
}
