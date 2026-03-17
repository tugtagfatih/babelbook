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

// BuildSystemPrompt creates the translation system instruction for the given language pair.
// If extraPrompt is non-empty, it is appended as an additional rule.
func BuildSystemPrompt(sourceLang, targetLang, extraPrompt string) string {
	prompt := fmt.Sprintf(`You are an award-winning literary translator and linguist. Your task is to translate the given %s text into %s.
Rules:
1. Produce fluent, natural translations that follow proper grammar rules.
2. You may receive text containing HTML tags (e.g. <b>, <i>). Do NOT alter the tag structure.
3. Return ONLY the translated text, do not add any comments.`, sourceLang, targetLang)

	if extraPrompt != "" {
		prompt += fmt.Sprintf("\n4. Additional instructions: %s", extraPrompt)
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
