// Package translator provides text translation via AI providers.
package translator

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tugtagfatih/babelbook/provider"
)

const (
	maxRetries = 3
	retryDelay = 3 * time.Second
)

// BuildSystemPrompt creates the translation system instruction.
// glossaryRules and extraPrompt are optional (pass "" to skip).
func BuildSystemPrompt(sourceLang, targetLang, glossaryRules, extraPrompt string) string {
	prompt := fmt.Sprintf(`You are an award-winning literary translator and linguist. Your task is to translate the given %s text into %s.
Rules:
1. Produce fluent, natural translations that follow proper grammar rules.
2. You may receive text containing HTML tags (e.g. <b>, <i>). Do NOT alter the tag structure.
3. Return ONLY the translated text, do not add any comments.`, sourceLang, targetLang)

	ruleNum := 4
	if glossaryRules != "" {
		prompt += fmt.Sprintf("\n%d. %s", ruleNum, glossaryRules)
		ruleNum++
	}
	if extraPrompt != "" {
		prompt += fmt.Sprintf("\n%d. Additional instructions: %s", ruleNum, extraPrompt)
	}
	return prompt
}

// Translate sends text to the AI provider for translation and returns the result.
func Translate(p *provider.Provider, apiURL, systemPrompt, text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	jsonData, err := p.BuildRequestBody(systemPrompt, text)
	if err != nil {
		fmt.Printf("  ✗ Failed to build request: %v\n", err)
		return text
	}

	client := &http.Client{Timeout: 300 * time.Second}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := p.BuildHTTPRequest(apiURL, jsonData)
		if err != nil {
			fmt.Printf("  ✗ Failed to create HTTP request: %v\n", err)
			return text
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			translated, parseErr := p.ParseResponse(resp.Body)
			resp.Body.Close()
			if parseErr == nil {
				return translated
			}
			fmt.Printf("  ✗ Parse error: %v\n", parseErr)
		} else if err != nil {
			fmt.Printf("  ✗ Request error (attempt %d/%d): %v\n", attempt, maxRetries, err)
		} else {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("  ✗ API returned status %d (attempt %d/%d)\n  Response: %s\n", resp.StatusCode, attempt, maxRetries, string(body))
			resp.Body.Close()
		}

		if attempt < maxRetries {
			fmt.Printf("  ⏳ Retrying in %v... (attempt %d/%d)\n", retryDelay, attempt+1, maxRetries)
			time.Sleep(retryDelay)
		}
	}
	fmt.Println("  ⚠ All retries exhausted, keeping original text")
	return text
}

// EstimateCost gives a rough cost estimate based on character count and provider.
func EstimateCost(totalChars int, providerName string) (inputTokens int, outputTokens int, costUSD float64) {
	// Rough token estimate: 1 token ≈ 4 characters
	inputTokens = totalChars / 4
	// Output is roughly same size as input for translation
	outputTokens = inputTokens

	// Pricing per 1M tokens (as of 2025)
	var inputPricePerM, outputPricePerM float64
	switch providerName {
	case "Gemini":
		// Gemini Flash models are very cheap / free tier
		inputPricePerM = 0.10
		outputPricePerM = 0.40
	case "OpenAI":
		// GPT-4o pricing
		inputPricePerM = 2.50
		outputPricePerM = 10.00
	case "Anthropic":
		// Claude Sonnet pricing
		inputPricePerM = 3.00
		outputPricePerM = 15.00
	default:
		inputPricePerM = 1.00
		outputPricePerM = 3.00
	}

	costUSD = (float64(inputTokens)/1_000_000)*inputPricePerM + (float64(outputTokens)/1_000_000)*outputPricePerM
	return
}
