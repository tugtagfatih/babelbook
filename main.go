package main

import (
	"bufio"
	"log"
	"os"

	"github.com/tugtagfatih/babelbook/config"
	"github.com/tugtagfatih/babelbook/epub"
	"github.com/tugtagfatih/babelbook/provider"
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
		// User chose custom from model menu → full custom flow
		customProvider, customModel, ok := ui.SelectCustomProvider(reader)
		if !ok {
			log.Fatal("Custom provider setup failed.")
		}
		selectedProvider = customProvider
		selectedModel = customModel
	}

	// Inject model name into request body for providers that need it
	provider.InjectModel(selectedProvider, selectedModel)

	// Find EPUB files in the current directory
	epubFiles, err := epub.FindFiles(".")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if len(epubFiles) == 0 {
		log.Println("No .epub files found in this directory.")
		return
	}

	// Interactive file and language selection
	inputFile, err := ui.SelectFile(epubFiles)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	sourceLang := ui.PromptLanguage(reader, "Source language (e.g. English)", "English")
	targetLang := ui.PromptLanguage(reader, "Target language (e.g. Turkish)", "Turkish")

	outputFile := epub.GenerateOutputFilename(targetLang, inputFile)
	systemPrompt := translator.BuildSystemPrompt(sourceLang, targetLang)

	ui.PrintStartInfo(selectedProvider.Name, selectedModel, sourceLang, inputFile, outputFile)
	ui.ConfirmStart(reader)

	// Translate the EPUB
	if err := epub.TranslateEPUB(inputFile, outputFile, selectedProvider, selectedModel, systemPrompt); err != nil {
		log.Fatal("Translation failed: ", err)
	}

	ui.PrintCompletion(outputFile)
	ui.PauseBeforeExit(reader)
}
