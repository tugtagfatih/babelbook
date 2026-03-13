// Package epub handles reading, translating, and writing EPUB files.
package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tugtagfatih/babelbook/provider"
	"github.com/tugtagfatih/babelbook/translator"
)

// translatableSelectors defines HTML elements whose text content should be translated.
const translatableSelectors = "p, h1, h2, h3, h4, li, span"

// rateLimitDelay is the pause between API calls to avoid rate limiting.
const rateLimitDelay = 1 * time.Second

// FindFiles scans the given directory and returns a list of .epub filenames.
func FindFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var epubFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".epub") {
			epubFiles = append(epubFiles, entry.Name())
		}
	}
	return epubFiles, nil
}

// GenerateOutputFilename creates an output filename by prefixing the target language code.
func GenerateOutputFilename(targetLang, inputFile string) string {
	if len(targetLang) >= 2 {
		prefix := strings.ToUpper(targetLang[:2])
		return fmt.Sprintf("%s_%s", prefix, inputFile)
	}
	return fmt.Sprintf("translated_%s", inputFile)
}

// isHTMLFile checks whether a filename has an HTML or XHTML extension.
func isHTMLFile(name string) bool {
	return strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".xhtml")
}

// countHTMLFiles counts the number of HTML/XHTML files in an EPUB archive.
func countHTMLFiles(files []*zip.File) int {
	count := 0
	for _, f := range files {
		if isHTMLFile(f.Name) {
			count++
		}
	}
	return count
}

// TranslateEPUB reads the input EPUB, translates all HTML content, and writes the result.
func TranslateEPUB(inputPath, outputPath string, p *provider.Provider, model, systemPrompt string) error {
	apiURL := p.BuildURL(model)

	reader, err := zip.OpenReader(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer reader.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	writer := zip.NewWriter(outputFile)
	defer writer.Close()

	totalHTML := countHTMLFiles(reader.File)
	processedHTML := 0

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in EPUB: %w", file.Name, err)
		}

		if isHTMLFile(file.Name) {
			processedHTML++
			fmt.Printf("\n📄 Processing [%d/%d]: %s\n", processedHTML, totalHTML, file.Name)

			if err := translateAndWriteHTML(rc, writer, file.Name, p, apiURL, systemPrompt); err != nil {
				rc.Close()
				return err
			}
		} else {
			fileWriter, err := writer.Create(file.Name)
			if err != nil {
				rc.Close()
				return fmt.Errorf("failed to create entry %s: %w", file.Name, err)
			}
			io.Copy(fileWriter, rc)
		}
		rc.Close()
	}
	return nil
}

// translateAndWriteHTML parses an HTML document, translates its content, and writes it to the zip.
func translateAndWriteHTML(rc io.Reader, writer *zip.Writer, name string, p *provider.Provider, apiURL, systemPrompt string) error {
	doc, err := goquery.NewDocumentFromReader(rc)
	if err != nil {
		return fmt.Errorf("failed to parse HTML %s: %w", name, err)
	}

	// Count translatable elements
	elements := doc.Find(translatableSelectors)
	totalElements := elements.Length()
	translatedCount := 0

	elements.Each(func(i int, s *goquery.Selection) {
		originalHTML, _ := s.Html()
		if strings.TrimSpace(originalHTML) != "" {
			translatedCount++
			// Truncate preview to 50 chars for display
			preview := strings.TrimSpace(originalHTML)
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			fmt.Printf("  🔄 Translating element %d/%d: %s\n", translatedCount, totalElements, preview)

			translatedHTML := translator.Translate(p, apiURL, systemPrompt, originalHTML)
			s.SetHtml(translatedHTML)

			fmt.Printf("  ⏳ Rate limit pause (%v)...\n", rateLimitDelay)
			time.Sleep(rateLimitDelay)
			fmt.Printf("  ✓ Element %d/%d done\n", translatedCount, totalElements)
		}
	})

	fmt.Printf("  ✅ File complete: %d elements translated\n", translatedCount)

	htmlStr, _ := doc.Html()
	fileWriter, err := writer.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create entry %s: %w", name, err)
	}
	fileWriter.Write([]byte(htmlStr))
	return nil
}
