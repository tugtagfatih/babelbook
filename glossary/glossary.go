// Package glossary provides term-to-translation mapping for consistent translations.
package glossary

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Entry represents a single glossary term mapping.
type Entry struct {
	Source string
	Target string
}

// Glossary holds the loaded term mappings.
type Glossary struct {
	Entries []Entry
}

// Load reads a glossary file. Each line should be: source -> target
// Lines starting with # are comments. Empty lines are skipped.
// Returns nil (not an error) if file does not exist.
func Load(path string) (*Glossary, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open glossary: %w", err)
	}
	defer file.Close()

	g := &Glossary{}
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "->", 2)
		if len(parts) != 2 {
			// Try with => as alternative separator
			parts = strings.SplitN(line, "=>", 2)
		}
		if len(parts) != 2 {
			fmt.Printf("  ⚠ Glossary line %d skipped (use 'source -> target'): %s\n", lineNum, line)
			continue
		}

		source := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(parts[1])
		if source != "" && target != "" {
			g.Entries = append(g.Entries, Entry{Source: source, Target: target})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading glossary: %w", err)
	}

	return g, nil
}

// ToPromptRules converts the glossary entries to a prompt instruction string.
func (g *Glossary) ToPromptRules() string {
	if g == nil || len(g.Entries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("GLOSSARY — You MUST translate these terms exactly as specified:\n")
	for _, e := range g.Entries {
		sb.WriteString(fmt.Sprintf("  \"%s\" → \"%s\"\n", e.Source, e.Target))
	}
	return sb.String()
}
