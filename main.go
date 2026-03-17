package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/tugtagfatih/babelbook/config"
	"github.com/tugtagfatih/babelbook/epub"
	"github.com/tugtagfatih/babelbook/provider"
	"github.com/tugtagfatih/babelbook/settings"
	"github.com/tugtagfatih/babelbook/translator"
	"github.com/tugtagfatih/babelbook/ui"
)

func main() {
	ui.PrintBanner()
	reader := bufio.NewReader(os.Stdin)

	// Load configuration and detect available providers
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// Load saved preferences (provider, model, settings)
	s := settings.Load()

	var selectedProvider *provider.Provider
	var selectedModel string

	// Try to restore saved provider/model
	if s.SavedProviderEnvKey != "" && s.SavedModel != "" {
		for _, p := range cfg.Providers {
			if p.EnvKey == s.SavedProviderEnvKey {
				selectedProvider = p
				selectedModel = s.SavedModel
				fmt.Printf("✓ Using saved provider: %s / %s\n", p.Name, selectedModel)
				break
			}
		}
	}

	// If no saved preference (first run), ask the user
	if selectedProvider == nil {
		selectedProvider = ui.SelectProvider(reader, cfg.Providers)
		selectedModel = ui.SelectModel(reader, selectedProvider)
		if selectedModel == "" {
			customProvider, customModel, ok := ui.SelectCustomProvider(reader)
			if !ok {
				log.Fatal("Custom provider setup failed.")
			}
			selectedProvider = customProvider
			selectedModel = customModel
		}

		// Save the choice for next time
		s.SavedProviderEnvKey = selectedProvider.EnvKey
		s.SavedModel = selectedModel
		s.Save()
	}

	// Inject model name into request body for providers that need it
	provider.InjectModel(selectedProvider, selectedModel)

	// Main menu loop: translate or settings
	for {
		choice := ui.ShowMainMenu(reader, s, selectedProvider.Name, selectedModel)
		if choice == "translate" {
			break
		}
		if choice == "change_model" {
			// Re-select provider and model
			selectedProvider = ui.SelectProvider(reader, cfg.Providers)
			selectedModel = ui.SelectModel(reader, selectedProvider)
			if selectedModel == "" {
				customProvider, customModel, ok := ui.SelectCustomProvider(reader)
				if !ok {
					log.Fatal("Custom provider setup failed.")
				}
				selectedProvider = customProvider
				selectedModel = customModel
			}
			provider.InjectModel(selectedProvider, selectedModel)
			s.SavedProviderEnvKey = selectedProvider.EnvKey
			s.SavedModel = selectedModel
			s.Save()
		}
	}

	// Find EPUB files
	epubFiles, err := epub.FindFiles(".")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if len(epubFiles) == 0 {
		log.Println("No .epub files found in this directory.")
		return
	}

	// File and language selection
	inputFile, err := ui.SelectFile(epubFiles)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	sourceLang := ui.PromptLanguage(reader, "Source language (e.g. English)", "English")
	targetLang := ui.PromptLanguage(reader, "Target language (e.g. Turkish)", "Turkish")

	outputFile := epub.GenerateOutputFilename(targetLang, inputFile, s.Bilingual)
	systemPrompt := translator.BuildSystemPrompt(sourceLang, targetLang, s.ExtraPrompt)

	ui.PrintStartInfo(selectedProvider.Name, selectedModel, sourceLang, inputFile, outputFile, s)
	ui.ConfirmStart(reader)

	// Translate the EPUB
	if err := epub.TranslateEPUB(inputFile, outputFile, selectedProvider, selectedModel, systemPrompt, s); err != nil {
		log.Fatal("Translation failed: ", err)
	}

	ui.PrintCompletion(outputFile)
	ui.PauseBeforeExit(reader)
}
