// Package provider defines AI translation providers and their API specifics.
package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Model represents an AI model offered by a provider.
type Model struct {
	Name      string
	IsDefault bool
}

// Provider represents an AI translation service with its API configuration.
type Provider struct {
	Name      string
	EnvKey    string
	APIKey    string
	Models    []Model
	buildURL  func(apiKey, model string) string
	buildReq  func(systemPrompt, text string) ([]byte, error)
	parseRes  func(body io.Reader) (string, error)
	buildHTTP func(apiURL, apiKey string, jsonData []byte) (*http.Request, error)
}

// BuildURL constructs the full API endpoint URL for the given model.
func (p *Provider) BuildURL(model string) string {
	return p.buildURL(p.APIKey, model)
}

// BuildRequestBody creates the JSON request body for the translation API call.
func (p *Provider) BuildRequestBody(systemPrompt, text string) ([]byte, error) {
	return p.buildReq(systemPrompt, text)
}

// ParseResponse extracts the translated text from the API response body.
func (p *Provider) ParseResponse(body io.Reader) (string, error) {
	return p.parseRes(body)
}

// BuildHTTPRequest creates a fully configured *http.Request for the provider's API.
func (p *Provider) BuildHTTPRequest(apiURL string, jsonData []byte) (*http.Request, error) {
	return p.buildHTTP(apiURL, p.APIKey, jsonData)
}

// DefaultModel returns the default model name for this provider.
func (p *Provider) DefaultModel() string {
	for _, m := range p.Models {
		if m.IsDefault {
			return m.Name
		}
	}
	if len(p.Models) > 0 {
		return p.Models[0].Name
	}
	return ""
}

// -------------------------------------------------------------------
// Gemini
// -------------------------------------------------------------------

func newGemini(apiKey string) *Provider {
	return &Provider{
		Name:   "Gemini",
		EnvKey: "GEMINI_API_KEY",
		APIKey: apiKey,
		Models: []Model{
			{Name: "gemini-3-flash-preview", IsDefault: true},
			{Name: "gemini-2.5-flash"},
			{Name: "gemini-2.5-pro"},
			{Name: "gemini-2.0-flash"},
		},
		buildURL: func(key, model string) string {
			return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, key)
		},
		buildReq: buildGeminiRequest,
		parseRes: parseGeminiResponse,
		buildHTTP: func(apiURL, _ string, jsonData []byte) (*http.Request, error) {
			// Gemini uses key in URL, no auth header needed
			return http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
		},
	}
}

type geminiRequest struct {
	SystemInstruction struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"system_instruction"`
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

func buildGeminiRequest(systemPrompt, text string) ([]byte, error) {
	req := geminiRequest{}
	req.SystemInstruction.Parts = append(req.SystemInstruction.Parts, struct {
		Text string `json:"text"`
	}{Text: systemPrompt})
	req.Contents = append(req.Contents, struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{
		Parts: []struct {
			Text string `json:"text"`
		}{{Text: text}},
	})
	return json.Marshal(req)
}

func parseGeminiResponse(body io.Reader) (string, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Gemini response: %w", err)
	}
	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates in Gemini response")
	}
	content, ok := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid content structure in Gemini response")
	}
	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", fmt.Errorf("no parts in Gemini response")
	}
	text, ok := parts[0].(map[string]interface{})["text"].(string)
	if !ok {
		return "", fmt.Errorf("invalid text in Gemini response")
	}
	return text, nil
}

// -------------------------------------------------------------------
// OpenAI
// -------------------------------------------------------------------

func newOpenAI(apiKey string) *Provider {
	return &Provider{
		Name:   "OpenAI",
		EnvKey: "OPENAI_API_KEY",
		APIKey: apiKey,
		Models: []Model{
			{Name: "gpt-4o", IsDefault: true},
			{Name: "gpt-4o-mini"},
			{Name: "o3-mini"},
		},
		buildURL: func(_, _ string) string {
			return "https://api.openai.com/v1/chat/completions"
		},
		buildReq: buildOpenAIRequest,
		parseRes: parseOpenAIResponse,
		buildHTTP: func(apiURL, key string, jsonData []byte) (*http.Request, error) {
			req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+key)
			req.Header.Set("Content-Type", "application/json")
			return req, nil
		},
	}
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAISelectedModel is set at runtime when the user picks a model.
// The buildReq closure captures the provider, but we need to pass the model.
// We'll handle this via a package-level approach or by wrapping.

func buildOpenAIRequest(systemPrompt, text string) ([]byte, error) {
	// Model will be injected by the Translate function via SetModel
	req := openAIRequest{
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: text},
		},
	}
	return json.Marshal(req)
}

func parseOpenAIResponse(body io.Reader) (string, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}
	message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message structure in OpenAI response")
	}
	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid content in OpenAI response")
	}
	return content, nil
}

// -------------------------------------------------------------------
// Anthropic
// -------------------------------------------------------------------

func newAnthropic(apiKey string) *Provider {
	return &Provider{
		Name:   "Anthropic",
		EnvKey: "ANTHROPIC_API_KEY",
		APIKey: apiKey,
		Models: []Model{
			{Name: "claude-sonnet-4-20250514", IsDefault: true},
			{Name: "claude-3-5-haiku-20241022"},
		},
		buildURL: func(_, _ string) string {
			return "https://api.anthropic.com/v1/messages"
		},
		buildReq: buildAnthropicRequest,
		parseRes: parseAnthropicResponse,
		buildHTTP: func(apiURL, key string, jsonData []byte) (*http.Request, error) {
			req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
			if err != nil {
				return nil, err
			}
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("Content-Type", "application/json")
			return req, nil
		},
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func buildAnthropicRequest(systemPrompt, text string) ([]byte, error) {
	req := anthropicRequest{
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: text},
		},
	}
	return json.Marshal(req)
}

func parseAnthropicResponse(body io.Reader) (string, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Anthropic response: %w", err)
	}
	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("no content in Anthropic response")
	}
	block, ok := content[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid content block in Anthropic response")
	}
	text, ok := block["text"].(string)
	if !ok {
		return "", fmt.Errorf("invalid text in Anthropic response")
	}
	return text, nil
}

// -------------------------------------------------------------------
// Registry: detect available providers from environment
// -------------------------------------------------------------------

// knownProviderKeys lists the env variable names and their constructor functions.
var knownProviderKeys = []struct {
	envKey      string
	constructor func(string) *Provider
}{
	{"GEMINI_API_KEY", newGemini},
	{"OPENAI_API_KEY", newOpenAI},
	{"ANTHROPIC_API_KEY", newAnthropic},
}

// DetectProviders reads the environment and returns providers that have API keys set.
func DetectProviders(getEnv func(string) string) []*Provider {
	var available []*Provider
	for _, kp := range knownProviderKeys {
		key := getEnv(kp.envKey)
		if key != "" {
			available = append(available, kp.constructor(key))
		}
	}
	return available
}

// NewCustomProvider creates a provider from a user-specified env key and model name.
// It tries to match the env key to a known provider; if not found, defaults to Gemini-style API.
func NewCustomProvider(envKey, apiKey, modelName string) *Provider {
	// Try to match to known provider
	for _, kp := range knownProviderKeys {
		if kp.envKey == envKey {
			p := kp.constructor(apiKey)
			p.Models = []Model{{Name: modelName, IsDefault: true}}
			return p
		}
	}
	// Fallback: treat as Gemini-compatible
	p := newGemini(apiKey)
	p.Name = "Custom"
	p.EnvKey = envKey
	p.Models = []Model{{Name: modelName, IsDefault: true}}
	return p
}

// InjectModel sets the model name into request bodies that need it (OpenAI, Anthropic).
// This wraps the original buildReq to inject the model field after marshaling.
func InjectModel(p *Provider, model string) {
	originalBuildReq := p.buildReq
	p.buildReq = func(systemPrompt, text string) ([]byte, error) {
		data, err := originalBuildReq(systemPrompt, text)
		if err != nil {
			return nil, err
		}
		// For providers that need a "model" field in the JSON body
		if p.Name == "OpenAI" || p.Name == "Anthropic" || p.Name == "Custom" {
			var raw map[string]interface{}
			json.Unmarshal(data, &raw)
			raw["model"] = model
			return json.Marshal(raw)
		}
		return data, nil
	}
}
