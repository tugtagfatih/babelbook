// Package config handles application configuration loading from environment variables.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/tugtagfatih/babelbook/provider"
)

// Config holds the application configuration.
type Config struct {
	Providers []*provider.Provider
}

// Load reads the .env file and returns available AI providers.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w\nPlease create a .env file (see .env.example)")
	}

	providers := provider.DetectProviders(os.Getenv)
	if len(providers) == 0 {
		return nil, fmt.Errorf("no API keys found in .env file\nPlease add at least one of: GEMINI_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY")
	}

	return &Config{
		Providers: providers,
	}, nil
}

// ReadEnvFile reads the raw .env file contents and returns all key=value lines.
func ReadEnvFile() ([]EnvLine, error) {
	data, err := os.ReadFile(".env")
	if err != nil {
		return nil, fmt.Errorf("failed to read .env file: %w", err)
	}

	var lines []EnvLine
	for _, line := range splitLines(string(data)) {
		key, value := parseEnvLine(line)
		if key != "" {
			lines = append(lines, EnvLine{Key: key, Value: value, Raw: line})
		}
	}
	return lines, nil
}

// EnvLine represents a single key=value entry in the .env file.
type EnvLine struct {
	Key   string
	Value string
	Raw   string
}

// splitLines splits text into lines handling both \n and \r\n.
func splitLines(text string) []string {
	var lines []string
	current := ""
	for _, ch := range text {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else if ch == '\r' {
			continue
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// parseEnvLine extracts key and value from a "KEY=VALUE" line.
func parseEnvLine(line string) (string, string) {
	// Skip comments and empty lines
	trimmed := ""
	for _, ch := range line {
		if ch == ' ' || ch == '\t' {
			continue
		}
		trimmed += string(ch)
		break
	}
	if trimmed == "#" || trimmed == "" {
		return "", ""
	}

	for i, ch := range line {
		if ch == '=' {
			return line[:i], line[i+1:]
		}
	}
	return "", ""
}
