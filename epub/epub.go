// Package epub handles reading, translating, and writing EPUB files.
package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/tugtagfatih/babelbook/provider"
	"github.com/tugtagfatih/babelbook/translator"
)

// =====================================================================
// Rate Limit Configuration
// =====================================================================
// These values are tuned for the Gemini free/paid tier.
// If you know your API rate limits, you can adjust these values:
//
//   maxChunkChars  - Characters per translation chunk. Increase for models
//                    with higher output token limits. Decrease if you get
//                    truncated translations. (Default: 50000 = ~12K tokens)
//
//   maxConcurrent  - Max parallel API requests at once. Should not exceed
//                    your RPM (Requests Per Minute) limit divided by ~3.
//                    Too high = 429 rate limit errors. (Default: 20)
//
// Current defaults are optimized for:
//   RPM (Requests Per Minute) : 1,000
//   TPM (Tokens Per Minute)   : 2,000,000
//   RPD (Requests Per Day)    : 10,000
// =====================================================================

// maxChunkChars is the maximum character count per translation chunk.
const maxChunkChars = 50000

// maxConcurrent is the maximum number of parallel API requests.
const maxConcurrent = 20

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

// splitBodyIntoChunks splits the body HTML into chunks at paragraph/heading boundaries.
// Each chunk stays under maxChunkChars characters.
func splitBodyIntoChunks(bodyHTML string) []string {
	// Split tags to use as boundaries
	splitTags := []string{"</p>", "</h1>", "</h2>", "</h3>", "</h4>", "</li>", "</div>"}

	var chunks []string
	remaining := bodyHTML

	for len(remaining) > 0 {
		if len(remaining) <= maxChunkChars {
			chunks = append(chunks, remaining)
			break
		}

		// Find the best split point: last closing tag before maxChunkChars
		bestSplit := -1
		for _, tag := range splitTags {
			// Search for the tag within the first maxChunkChars characters
			searchArea := remaining[:maxChunkChars]
			idx := strings.LastIndex(searchArea, tag)
			if idx > bestSplit {
				bestSplit = idx + len(tag)
			}
		}

		if bestSplit <= 0 {
			// No good split point found, force split at maxChunkChars
			bestSplit = maxChunkChars
		}

		chunks = append(chunks, remaining[:bestSplit])
		remaining = remaining[bestSplit:]
	}

	return chunks
}

// translateAndWriteHTML splits the body into chapter-sized chunks and translates them concurrently.
func translateAndWriteHTML(rc io.Reader, writer *zip.Writer, name string, p *provider.Provider, apiURL, systemPrompt string) error {
	doc, err := goquery.NewDocumentFromReader(rc)
	if err != nil {
		return fmt.Errorf("failed to parse HTML %s: %w", name, err)
	}

	body := doc.Find("body")
	if body.Length() == 0 {
		fmt.Println("  ⚠ No <body> tag found, skipping")
		htmlStr, _ := doc.Html()
		fw, _ := writer.Create(name)
		fw.Write([]byte(htmlStr))
		return nil
	}

	bodyHTML, _ := body.Html()
	totalChars := len(bodyHTML)

	if strings.TrimSpace(bodyHTML) == "" {
		fmt.Println("  ⚠ Empty body, skipping")
		htmlStr, _ := doc.Html()
		fw, _ := writer.Create(name)
		fw.Write([]byte(htmlStr))
		return nil
	}

	// Split into chunks
	chunks := splitBodyIntoChunks(bodyHTML)
	totalChunks := len(chunks)

	fmt.Printf("  📊 Total: %d characters → split into %d chunks\n", totalChars, totalChunks)
	fmt.Printf("  🚀 Sending %d chunks concurrently (max %d parallel)...\n", totalChunks, maxConcurrent)

	// Translate all chunks concurrently
	translatedChunks := make([]string, totalChunks)
	var mu sync.Mutex
	var wg sync.WaitGroup
	completed := 0

	sem := make(chan struct{}, maxConcurrent)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, chunkHTML string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			chunkLen := len(chunkHTML)
			translated := translator.Translate(p, apiURL, systemPrompt, chunkHTML)

			mu.Lock()
			translatedChunks[idx] = translated
			completed++
			fmt.Printf("  ✓ Chunk %d/%d done (%d chars)\n", completed, totalChunks, chunkLen)
			mu.Unlock()
		}(i, chunk)
	}

	wg.Wait()

	// Reassemble translated body
	translatedBody := strings.Join(translatedChunks, "")
	body.SetHtml(translatedBody)

	fmt.Printf("  ✅ File complete: %d chunks translated\n", totalChunks)

	htmlStr, _ := doc.Html()
	fileWriter, err := writer.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create entry %s: %w", name, err)
	}
	fileWriter.Write([]byte(htmlStr))
	return nil
}
