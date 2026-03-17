// Package settings holds runtime configuration for the translation engine.
package settings

// Settings contains user-configurable options displayed in the TUI settings menu.
type Settings struct {
	// MaxChunkChars is the maximum character count per translation chunk.
	MaxChunkChars int

	// MaxConcurrent is the maximum number of parallel API requests.
	MaxConcurrent int

	// Bilingual enables dual-language output (original + translated).
	Bilingual bool

	// ExtraPrompt is an optional custom instruction appended to the system prompt.
	ExtraPrompt string
}

// Default returns the default settings optimized for:
//
//	RPM: 1,000 | TPM: 2,000,000 | RPD: 10,000
func Default() *Settings {
	return &Settings{
		MaxChunkChars: 50000,
		MaxConcurrent: 20,
		Bilingual:     false,
		ExtraPrompt:   "",
	}
}
