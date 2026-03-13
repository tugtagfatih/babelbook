// Package ui provides CLI interaction helpers for the EPUB translator.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tugtagfatih/babelbook/config"
	"github.com/tugtagfatih/babelbook/provider"
)

// PrintBanner displays the application header.
func PrintBanner() {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     EPUB Translation Engine          ║")
	fmt.Println("╚══════════════════════════════════════╝")
}

// SelectProvider prompts the user to choose an AI provider.
// If only one is available, it auto-selects it.
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
// The default model is pre-selected if the user presses Enter.
// Includes a "Custom" option to enter a model name from .env.
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

	// Custom entry
	if idx == customIdx {
		return selectCustomModel(reader)
	}

	if idx > 0 && idx <= len(p.Models) {
		fmt.Printf("✓ Selected model: %s\n", p.Models[idx-1].Name)
		return p.Models[idx-1].Name
	}

	// Invalid → use default
	defaultModel := p.DefaultModel()
	fmt.Printf("Invalid selection, using default: %s\n", defaultModel)
	return defaultModel
}

// selectCustomModel lets the user pick an API key from .env and enter a custom model name.
// It returns the model name and updates the provider in-place via the returned provider.
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

// SelectCustomProvider guides the user through selecting an API key from .env
// and creating a custom provider.
func SelectCustomProvider(reader *bufio.Reader) (*provider.Provider, string, bool) {
	envLines, err := config.ReadEnvFile()
	if err != nil {
		fmt.Printf("✗ Could not read .env file: %v\n", err)
		return nil, "", false
	}

	// Show only lines that look like API keys
	var keyLines []config.EnvLine
	for _, line := range envLines {
		if strings.Contains(strings.ToUpper(line.Key), "KEY") || strings.Contains(strings.ToUpper(line.Key), "API") {
			keyLines = append(keyLines, line)
		}
	}
	if len(keyLines) == 0 {
		keyLines = envLines // Show all if no KEY/API lines found
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

// maskValue shows only the first 4 and last 4 characters of a value.
func maskValue(val string) string {
	if val == "" {
		return "(empty)"
	}
	if len(val) <= 8 {
		return "****"
	}
	return val[:4] + "..." + val[len(val)-4:]
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
func PrintStartInfo(providerName, model, sourceLang, inputFile, outputFile string) {
	fmt.Println("\n--------------------------------------------------")
	fmt.Printf("  Provider : %s\n", providerName)
	fmt.Printf("  Model    : %s\n", model)
	fmt.Printf("  Source   : %s\n", sourceLang)
	fmt.Printf("  File     : %s -> %s\n", inputFile, outputFile)
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
