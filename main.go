package main

import (
	"bufio"
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

	// Provider selection
	selectedProvider := ui.SelectProvider(reader, cfg.Providers)

	// Model selection (includes custom option)
	selectedModel := ui.SelectModel(reader, selectedProvider)
	if selectedModel == "" {
		customProvider, customModel, ok := ui.SelectCustomProvider(reader)
		if !ok {
			log.Fatal("Custom provider setup failed.")
		}
		selectedProvider = customProvider
		selectedModel = customModel
	}

	// Inject model name into request body for providers that need it
	provider.InjectModel(selectedProvider, selectedModel)

	// Initialize default settings
	s := settings.Default()

	// Main menu loop: translate or settings
	for {
		choice := ui.ShowMainMenu(reader, s)
		if choice == "translate" {
			break
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
