// Package ui provides CLI interaction helpers for the EPUB translator.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tugtagfatih/babelbook/config"
	"github.com/tugtagfatih/babelbook/provider"
	"github.com/tugtagfatih/babelbook/settings"
)

// PrintBanner displays the application header.
func PrintBanner() {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     EPUB Translation Engine          ║")
	fmt.Println("╚══════════════════════════════════════╝")
}

// SelectProvider prompts the user to choose an AI provider.
func SelectProvider(reader *bufio.Reader, providers []*provider.Provider) *provider.Provider {
	if len(providers) == 1 {
		fmt.Printf("✓ Provider auto-selected: %s\n", providers[0].Name)
		return providers[0]
	}

	fmt.Println("\nAvailable AI providers:")
	for i, p := range providers {
		fmt.Printf("  [%d] %s (%s)\n", i+1, p.Name, p.EnvKey)
	}

	var idx int
	for {
		fmt.Print("Select provider: ")
		var input string
		fmt.Scanln(&input)
		fmt.Sscanf(input, "%d", &idx)
		if idx > 0 && idx <= len(providers) {
			break
		}
		fmt.Println("Invalid selection, please enter a number from the list.")
	}
	selected := providers[idx-1]
	fmt.Printf("✓ Selected provider: %s\n", selected.Name)
	return selected
}

// SelectModel prompts the user to choose a model from the provider's list.
func SelectModel(reader *bufio.Reader, p *provider.Provider) string {
	fmt.Printf("\nAvailable models for %s:\n", p.Name)
	for i, m := range p.Models {
		defaultMark := ""
		if m.IsDefault {
			defaultMark = " [default]"
		}
		fmt.Printf("  [%d] %s%s\n", i+1, m.Name, defaultMark)
	}
	customIdx := len(p.Models) + 1
	fmt.Printf("  [%d] Custom: Use another API key / model from .env\n", customIdx)

	fmt.Printf("Model number [Default: 1]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		defaultModel := p.DefaultModel()
		fmt.Printf("✓ Selected model: %s\n", defaultModel)
		return defaultModel
	}

	var idx int
	fmt.Sscanf(input, "%d", &idx)

	if idx == customIdx {
		return selectCustomModel(reader)
	}

	if idx > 0 && idx <= len(p.Models) {
		fmt.Printf("✓ Selected model: %s\n", p.Models[idx-1].Name)
		return p.Models[idx-1].Name
	}

	defaultModel := p.DefaultModel()
	fmt.Printf("Invalid selection, using default: %s\n", defaultModel)
	return defaultModel
}

func selectCustomModel(reader *bufio.Reader) string {
	fmt.Println("\n--- Custom Model Entry ---")
	fmt.Println("Enter the model name (e.g. gemini-2.5-pro, gpt-4-turbo, claude-3-opus-20240229):")
	fmt.Print("Model name: ")
	modelName, _ := reader.ReadString('\n')
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		fmt.Println("No model name entered, aborting custom entry.")
		return ""
	}
	return modelName
}

// SelectCustomProvider guides the user through creating a custom provider.
func SelectCustomProvider(reader *bufio.Reader) (*provider.Provider, string, bool) {
	envLines, err := config.ReadEnvFile()
	if err != nil {
		fmt.Printf("✗ Could not read .env file: %v\n", err)
		return nil, "", false
	}

	var keyLines []config.EnvLine
	for _, line := range envLines {
		if strings.Contains(strings.ToUpper(line.Key), "KEY") || strings.Contains(strings.ToUpper(line.Key), "API") {
			keyLines = append(keyLines, line)
		}
	}
	if len(keyLines) == 0 {
		keyLines = envLines
	}

	fmt.Println("\nAPI keys found in .env:")
	for i, line := range keyLines {
		maskedValue := maskValue(line.Value)
		fmt.Printf("  [%d] %s = %s\n", i+1, line.Key, maskedValue)
	}

	var idx int
	for {
		fmt.Print("Select API key to use: ")
		var input string
		fmt.Scanln(&input)
		fmt.Sscanf(input, "%d", &idx)
		if idx > 0 && idx <= len(keyLines) {
			break
		}
		fmt.Println("Invalid selection.")
	}

	selectedLine := keyLines[idx-1]
	if selectedLine.Value == "" {
		fmt.Printf("✗ %s is empty. Please add a value in .env and restart.\n", selectedLine.Key)
		return nil, "", false
	}

	fmt.Print("Enter model name: ")
	modelName, _ := reader.ReadString('\n')
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		fmt.Println("No model name entered.")
		return nil, "", false
	}

	p := provider.NewCustomProvider(selectedLine.Key, selectedLine.Value, modelName)
	fmt.Printf("✓ Custom provider: %s with model %s\n", p.Name, modelName)
	return p, modelName, true
}

func maskValue(val string) string {
	if val == "" {
		return "(empty)"
	}
	if len(val) <= 8 {
		return "****"
	}
	return val[:4] + "..." + val[len(val)-4:]
}

// ShowMainMenu displays the main menu and returns the choice.
// Returns "translate" or "settings".
func ShowMainMenu(reader *bufio.Reader, s *settings.Settings) string {
	for {
		fmt.Println("\n┌──────────────────────────────┐")
		fmt.Println("│  Main Menu                   │")
		fmt.Println("├──────────────────────────────┤")
		fmt.Println("│  [1] 📖 Start Translation    │")
		fmt.Println("│  [2] ⚙  Settings             │")
		fmt.Println("└──────────────────────────────┘")

		fmt.Print("Select option [Default: 1]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "", "1":
			return "translate"
		case "2":
			ShowSettingsMenu(reader, s)
		default:
			fmt.Println("Invalid option.")
		}
	}
}

// ShowSettingsMenu displays and allows editing of translation settings.
func ShowSettingsMenu(reader *bufio.Reader, s *settings.Settings) {
	for {
		bilingualStr := "OFF"
		if s.Bilingual {
			bilingualStr = "ON"
		}
		promptStr := "(none)"
		if s.ExtraPrompt != "" {
			display := s.ExtraPrompt
			if len(display) > 40 {
				display = display[:40] + "..."
			}
			promptStr = display
		}

		fmt.Println("\n┌──────────────────────────────────────────┐")
		fmt.Println("│  ⚙ Settings                              │")
		fmt.Println("├──────────────────────────────────────────┤")
		fmt.Printf("│  [1] Chunk size      : %-18d│\n", s.MaxChunkChars)
		fmt.Printf("│  [2] Max parallel    : %-18d│\n", s.MaxConcurrent)
		fmt.Printf("│  [3] Bilingual mode  : %-18s│\n", bilingualStr)
		fmt.Printf("│  [4] Extra prompt    : %-18s│\n", promptStr)
		fmt.Println("│  [0] ← Back                              │")
		fmt.Println("└──────────────────────────────────────────┘")

		fmt.Print("Select option: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "0", "":
			return
		case "1":
			fmt.Printf("Current: %d chars. Enter new value: ", s.MaxChunkChars)
			val, _ := reader.ReadString('\n')
			val = strings.TrimSpace(val)
			var n int
			if _, err := fmt.Sscanf(val, "%d", &n); err == nil && n > 0 {
				s.MaxChunkChars = n
				fmt.Printf("✓ Chunk size set to %d\n", n)
			} else if val != "" {
				fmt.Println("Invalid value, keeping current.")
			}
		case "2":
			fmt.Printf("Current: %d requests. Enter new value: ", s.MaxConcurrent)
			val, _ := reader.ReadString('\n')
			val = strings.TrimSpace(val)
			var n int
			if _, err := fmt.Sscanf(val, "%d", &n); err == nil && n > 0 {
				s.MaxConcurrent = n
				fmt.Printf("✓ Max parallel set to %d\n", n)
			} else if val != "" {
				fmt.Println("Invalid value, keeping current.")
			}
		case "3":
			s.Bilingual = !s.Bilingual
			state := "OFF"
			if s.Bilingual {
				state = "ON"
			}
			fmt.Printf("✓ Bilingual mode: %s\n", state)
		case "4":
			fmt.Println("Enter extra prompt (e.g. 'Don't translate character names, use formal tone'):")
			fmt.Println("(Enter empty line to clear)")
			fmt.Print("> ")
			val, _ := reader.ReadString('\n')
			val = strings.TrimSpace(val)
			s.ExtraPrompt = val
			if val == "" {
				fmt.Println("✓ Extra prompt cleared")
			} else {
				fmt.Printf("✓ Extra prompt set: %s\n", val)
			}
		default:
			fmt.Println("Invalid option.")
		}
	}
}

// SelectFile displays a numbered list of files and prompts the user to choose one.
func SelectFile(files []string) (string, error) {
	fmt.Println("\nSelect a file to translate:")
	for i, file := range files {
		fmt.Printf("  [%d] %s\n", i+1, file)
	}

	var fileIndex int
	for {
		fmt.Print("File number: ")
		var input string
		fmt.Scanln(&input)
		fmt.Sscanf(input, "%d", &fileIndex)
		if fileIndex > 0 && fileIndex <= len(files) {
			break
		}
		fmt.Println("Invalid selection, please enter a number from the list.")
	}
	return files[fileIndex-1], nil
}

// PromptLanguage asks the user for a language name with a default fallback.
func PromptLanguage(reader *bufio.Reader, prompt, defaultVal string) string {
	fmt.Printf("%s [Default: %s]: ", prompt, defaultVal)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// PrintStartInfo displays the translation parameters before processing begins.
func PrintStartInfo(providerName, model, sourceLang, inputFile, outputFile string, s *settings.Settings) {
	bilingualStr := "No"
	if s.Bilingual {
		bilingualStr = "Yes (dual-language)"
	}
	fmt.Println("\n--------------------------------------------------")
	fmt.Printf("  Provider   : %s\n", providerName)
	fmt.Printf("  Model      : %s\n", model)
	fmt.Printf("  Source     : %s\n", sourceLang)
	fmt.Printf("  File       : %s -> %s\n", inputFile, outputFile)
	fmt.Printf("  Bilingual  : %s\n", bilingualStr)
	fmt.Printf("  Chunk size : %d chars\n", s.MaxChunkChars)
	fmt.Printf("  Parallel   : %d requests\n", s.MaxConcurrent)
	if s.ExtraPrompt != "" {
		fmt.Printf("  Extra      : %s\n", s.ExtraPrompt)
	}
	fmt.Println("--------------------------------------------------")
}

// PrintCompletion displays the success message after translation finishes.
func PrintCompletion(outputFile string) {
	fmt.Println("\n==================================================")
	fmt.Println("🎉 Translation completed successfully!")
	fmt.Printf("   Output: %s\n", outputFile)
	fmt.Println("==================================================")
}

// ConfirmStart waits for the user to press Enter before starting.
func ConfirmStart(reader *bufio.Reader) {
	fmt.Print("\nPress Enter to start translation...")
	reader.ReadString('\n')
}

// PauseBeforeExit keeps the terminal open until the user presses Enter.
func PauseBeforeExit(reader *bufio.Reader) {
	fmt.Print("\nPress Enter to exit...")
	reader.ReadString('\n')
	os.Exit(0)
}
