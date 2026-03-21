package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tugtagfatih/babelbook/config"
	"github.com/tugtagfatih/babelbook/epub"
	"github.com/tugtagfatih/babelbook/glossary"
	"github.com/tugtagfatih/babelbook/provider"
	"github.com/tugtagfatih/babelbook/settings"
	"github.com/tugtagfatih/babelbook/translator"
	"github.com/tugtagfatih/babelbook/ui"
)

func main() {
	// CLI Flags
	inputFlag := flag.String("input", "", "Input EPUB file path")
	targetFlag := flag.String("target", "", "Target language (e.g. Turkish)")
	sourceFlag := flag.String("source", "", "Source language (default: English)")
	bilingualFlag := flag.Bool("bilingual", false, "Generate bilingual EPUB")
	glossaryFlag := flag.String("glossary", "", "Path to glossary file")
	batchFlag := flag.Bool("batch", false, "Translate all EPUB files in current directory")
	flag.Parse()

	cliMode := *inputFlag != "" || *batchFlag

	ui.PrintBanner()
	reader := bufio.NewReader(os.Stdin)

	// Load config and settings
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error: ", err)
	}
	s := settings.Load()

	if *bilingualFlag {
		s.Bilingual = true
	}

	// Provider/model selection (saved or first-run)
	var selectedProvider *provider.Provider
	var selectedModel string

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
		s.SavedProviderEnvKey = selectedProvider.EnvKey
		s.SavedModel = selectedModel
		s.Save()
	}

	provider.InjectModel(selectedProvider, selectedModel)

	// Load glossary
	glossaryPath := *glossaryFlag
	if glossaryPath == "" {
		if _, err := os.Stat("glossary.txt"); err == nil {
			glossaryPath = "glossary.txt"
		}
	}
	var glossaryRules string
	if glossaryPath != "" {
		g, err := glossary.Load(glossaryPath)
		if err != nil {
			log.Printf("⚠ Glossary error: %v\n", err)
		} else if g != nil && len(g.Entries) > 0 {
			glossaryRules = g.ToPromptRules()
			fmt.Printf("✓ Glossary loaded: %d terms from %s\n", len(g.Entries), glossaryPath)
		}
	}

	// CLI Mode
	if cliMode {
		runCLI(selectedProvider, selectedModel, s, glossaryRules, *inputFlag, *sourceFlag, *targetFlag, *batchFlag)
		return
	}

	// TUI Mode — main menu
	for {
		choice := ui.ShowMainMenu(reader, s, selectedProvider.Name, selectedModel)
		if choice == "translate" {
			break
		}
		if choice == "change_model" {
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

	selectedFiles, batch := ui.SelectFileOrBatch(reader, epubFiles)
	if len(selectedFiles) == 0 {
		return
	}

	// Language selection with memory
	sourceDefault := "English"
	if s.SavedSourceLang != "" {
		sourceDefault = s.SavedSourceLang
	}
	targetDefault := "Turkish"
	if s.SavedTargetLang != "" {
		targetDefault = s.SavedTargetLang
	}

	sourceLang := ui.PromptLanguage(reader, "Source language", sourceDefault)
	targetLang := ui.PromptLanguage(reader, "Target language", targetDefault)

	// Save language choices
	s.SavedSourceLang = sourceLang
	s.SavedTargetLang = targetLang
	s.Save()

	for i, inputFile := range selectedFiles {
		if batch {
			fmt.Printf("\n📚 Book %d/%d: %s\n", i+1, len(selectedFiles), inputFile)
		}

		// Chapter selection (partial translation)
		var skipChapters map[string]bool
		if !batch {
			chapters, err := epub.ListChapters(inputFile)
			if err == nil && len(chapters) > 1 {
				skipChapters = ui.SelectChapters(reader, chapters)
			}
		}

		outputFile := epub.GenerateOutputFilename(targetLang, inputFile, s.Bilingual)
		systemPrompt := translator.BuildSystemPrompt(sourceLang, targetLang, glossaryRules, s.ExtraPrompt)

		showCostEstimate(inputFile, selectedProvider.Name, s, skipChapters)

		if !batch || i == 0 {
			ui.PrintStartInfo(selectedProvider.Name, selectedModel, sourceLang, inputFile, outputFile, s)
			ui.ConfirmStart(reader)
		}

		if err := epub.TranslateEPUB(inputFile, outputFile, selectedProvider, selectedModel, systemPrompt, s, skipChapters); err != nil {
			fmt.Printf("✗ Translation failed for %s: %v\n", inputFile, err)
			continue
		}
		ui.PrintCompletion(outputFile)
	}

	ui.PauseBeforeExit(reader)
}

func runCLI(p *provider.Provider, model string, s *settings.Settings, glossaryRules, inputFile, sourceLang, targetLang string, batch bool) {
	if sourceLang == "" {
		sourceLang = "English"
	}
	if targetLang == "" {
		targetLang = "Turkish"
	}

	var files []string
	if batch {
		var err error
		files, err = epub.FindFiles(".")
		if err != nil {
			log.Fatal("Error: ", err)
		}
		if len(files) == 0 {
			log.Println("No .epub files found.")
			return
		}
		fmt.Printf("📚 Batch mode: %d files found\n", len(files))
	} else {
		files = []string{inputFile}
	}

	for i, f := range files {
		if len(files) > 1 {
			fmt.Printf("\n📚 [%d/%d] %s\n", i+1, len(files), f)
		}
		outputFile := epub.GenerateOutputFilename(targetLang, f, s.Bilingual)
		systemPrompt := translator.BuildSystemPrompt(sourceLang, targetLang, glossaryRules, s.ExtraPrompt)

		showCostEstimate(f, p.Name, s, nil)

		if err := epub.TranslateEPUB(f, outputFile, p, model, systemPrompt, s, nil); err != nil {
			fmt.Printf("✗ Failed: %s — %v\n", f, err)
			continue
		}
		fmt.Printf("✅ Done: %s\n", outputFile)
	}
}

func showCostEstimate(inputFile, providerName string, s *settings.Settings, skipChapters map[string]bool) {
	totalChars, totalChunks, err := epub.AnalyzeEPUB(inputFile, s.MaxChunkChars, skipChapters)
	if err != nil {
		return
	}

	inputTokens, _, costUSD := translator.EstimateCost(totalChars, providerName)

	fmt.Printf("  💰 Estimate: ~%dk tokens, %d chunks", inputTokens/1000, totalChunks)
	if providerName == "Gemini" {
		fmt.Printf(" (free tier / ~$%.4f)\n", costUSD)
	} else if providerName == "Local AI" {
		fmt.Printf(" (local — free)\n")
	} else {
		fmt.Printf(" (~$%.2f)\n", costUSD)
	}
}
