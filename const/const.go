package _const

// ProviderType represents the type of AI provider
type ProviderType uint

const (
	ProviderOpenAICompatibility ProviderType = 0
)

var (
	TagNoData       = []byte{58}
	TagData         = []byte("data: ")
	TagDataDone     = []byte("[DONE]")
	TagPromptTokens = []byte("prompt_tokens")
)
