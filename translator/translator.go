// Package translator provides text translation via AI providers.
package translator

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tugtagfatih/babelbook/provider"
)

const (
	maxRetries = 3
	retryDelay = 3 * time.Second
)

// BuildSystemPrompt creates the translation system instruction for the given language pair.
func BuildSystemPrompt(sourceLang, targetLang string) string {
	return fmt.Sprintf(`You are an award-winning literary translator and linguist. Your task is to translate the given %s text into %s.
Rules:
1. Produce fluent, natural translations that follow proper grammar rules.
2. You may receive text containing HTML tags (e.g. <b>, <i>). Do NOT alter the tag structure.
3. Return ONLY the translated text, do not add any comments.`, sourceLang, targetLang)
}

// Translate sends text to the AI provider for translation and returns the result.
// On failure after retries, it returns the original text unchanged.
// Logs progress including retry attempts and wait times to the console.
func Translate(p *provider.Provider, apiURL, systemPrompt, text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	jsonData, err := p.BuildRequestBody(systemPrompt, text)
	if err != nil {
		fmt.Printf("  ✗ Failed to build request: %v\n", err)
		return text
	}

	client := &http.Client{Timeout: 60 * time.Second}

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
			fmt.Printf("  ✗ API returned status %d (attempt %d/%d)\n", resp.StatusCode, attempt, maxRetries)
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
