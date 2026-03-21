// Package settings holds runtime configuration for the translation engine.
package settings

import (
	"encoding/json"
	"os"
)

const prefsFile = ".babelbook_prefs.json"

// Settings contains user-configurable options displayed in the TUI settings menu.
type Settings struct {
	// MaxChunkChars is the maximum character count per translation chunk.
	MaxChunkChars int `json:"max_chunk_chars"`

	// MaxConcurrent is the maximum number of parallel API requests.
	MaxConcurrent int `json:"max_concurrent"`

	// Bilingual enables dual-language output (original + translated).
	Bilingual bool `json:"bilingual"`

	// ExtraPrompt is an optional custom instruction appended to the system prompt.
	ExtraPrompt string `json:"extra_prompt"`

	// SavedProviderEnvKey is the env key of the last used provider (e.g. "GEMINI_API_KEY").
	SavedProviderEnvKey string `json:"saved_provider_env_key"`

	// SavedModel is the last used model name (e.g. "gemini-3-flash-preview").
	SavedModel string `json:"saved_model"`

	// SavedSourceLang is the last used source language.
	SavedSourceLang string `json:"saved_source_lang"`

	// SavedTargetLang is the last used target language.
	SavedTargetLang string `json:"saved_target_lang"`
}

// Default returns the default settings.
func Default() *Settings {
	return &Settings{
		MaxChunkChars: 50000,
		MaxConcurrent: 20,
		Bilingual:     false,
		ExtraPrompt:   "",
	}
}

// Load reads saved preferences from disk. Falls back to defaults for missing fields.
func Load() *Settings {
	s := Default()
	data, err := os.ReadFile(prefsFile)
	if err != nil {
		return s
	}
	json.Unmarshal(data, s)
	// Ensure sane minimums
	if s.MaxChunkChars <= 0 {
		s.MaxChunkChars = 50000
	}
	if s.MaxConcurrent <= 0 {
		s.MaxConcurrent = 20
	}
	return s
}

// Save writes current preferences to disk.
func (s *Settings) Save() {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(prefsFile, data, 0o644)
}
