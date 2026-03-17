// Package epub handles reading, translating, and writing EPUB files.
package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/tugtagfatih/babelbook/cache"
	"github.com/tugtagfatih/babelbook/provider"
	"github.com/tugtagfatih/babelbook/settings"
	"github.com/tugtagfatih/babelbook/translator"
)

// translatableSelectors defines HTML elements whose text content should be translated.
const translatableSelectors = "p, h1, h2, h3, h4, li, span"

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
// For bilingual mode, it uses "BI_" as prefix.
func GenerateOutputFilename(targetLang, inputFile string, bilingual bool) string {
	if bilingual {
		return fmt.Sprintf("BI_%s", inputFile)
	}
	if len(targetLang) >= 2 {
		prefix := strings.ToUpper(targetLang[:2])
		return fmt.Sprintf("%s_%s", prefix, inputFile)
	}
	return fmt.Sprintf("translated_%s", inputFile)
}

// isHTMLFile checks whether a filename has an HTML or XHTML extension.
func isHTMLFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".xhtml")
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

// sanitizePath normalizes file paths with special characters.
func sanitizePath(name string) string {
	parts := strings.Split(name, "/")
	for i, part := range parts {
		// Decode any existing percent-encoding, then re-encode safely
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		// Only encode characters that are genuinely problematic
		safe := ""
		for _, ch := range decoded {
			if ch > 127 || ch == '#' || ch == '?' || ch == '[' || ch == ']' {
				safe += url.PathEscape(string(ch))
			} else {
				safe += string(ch)
			}
		}
		parts[i] = safe
	}
	return strings.Join(parts, "/")
}

// TranslateEPUB reads the input EPUB, translates all HTML content, and writes the result.
func TranslateEPUB(inputPath, outputPath string, p *provider.Provider, model, systemPrompt string, s *settings.Settings) error {
	apiURL := p.BuildURL(model)
	cacheDir := cache.Dir(inputPath)

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

	// Build a mapping of original → sanitized file names
	nameMap := make(map[string]string)
	for _, file := range reader.File {
		sanitized := sanitizePath(file.Name)
		if sanitized != file.Name {
			nameMap[file.Name] = sanitized
		}
	}

	totalHTML := countHTMLFiles(reader.File)
	processedHTML := 0

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in EPUB: %w", file.Name, err)
		}

		// Use sanitized name for output
		outName := file.Name
		if sanitized, ok := nameMap[file.Name]; ok {
			outName = sanitized
		}

		if isHTMLFile(file.Name) {
			processedHTML++
			fmt.Printf("\n📄 Processing [%d/%d]: %s\n", processedHTML, totalHTML, file.Name)

			if err := translateAndWriteHTML(rc, writer, outName, file.Name, p, apiURL, systemPrompt, s, cacheDir, nameMap); err != nil {
				rc.Close()
				return err
			}
		} else {
			fileWriter, err := writer.Create(outName)
			if err != nil {
				rc.Close()
				return fmt.Errorf("failed to create entry %s: %w", outName, err)
			}
			io.Copy(fileWriter, rc)
		}
		rc.Close()
	}

	// Clean cache after successful completion
	cache.Clean(cacheDir)
	fmt.Println("\n🗑  Cache cleaned.")

	return nil
}

// fixInternalLinks updates src/href attributes in HTML to match sanitized filenames.
func fixInternalLinks(doc *goquery.Document, nameMap map[string]string) {
	for original, sanitized := range nameMap {
		// Only fix the basename part
		origBase := filepath.Base(original)
		sanitizedBase := filepath.Base(sanitized)
		if origBase == sanitizedBase {
			continue
		}

		doc.Find("[src], [href]").Each(func(_ int, sel *goquery.Selection) {
			for _, attr := range []string{"src", "href"} {
				val, exists := sel.Attr(attr)
				if exists && strings.Contains(val, origBase) {
					newVal := strings.Replace(val, origBase, sanitizedBase, 1)
					sel.SetAttr(attr, newVal)
				}
			}
		})
	}
}

// splitBodyIntoChunks splits the body HTML into chunks at paragraph/heading boundaries.
func splitBodyIntoChunks(bodyHTML string, maxChunkChars int) []string {
	splitTags := []string{"</p>", "</h1>", "</h2>", "</h3>", "</h4>", "</li>", "</div>"}

	var chunks []string
	remaining := bodyHTML

	for len(remaining) > 0 {
		if len(remaining) <= maxChunkChars {
			chunks = append(chunks, remaining)
			break
		}

		bestSplit := -1
		for _, tag := range splitTags {
			searchArea := remaining[:maxChunkChars]
			idx := strings.LastIndex(searchArea, tag)
			if idx > bestSplit {
				bestSplit = idx + len(tag)
			}
		}

		if bestSplit <= 0 {
			bestSplit = maxChunkChars
		}

		chunks = append(chunks, remaining[:bestSplit])
		remaining = remaining[bestSplit:]
	}

	return chunks
}

// translateAndWriteHTML splits the body into chunks and translates them concurrently.
// Supports resume from cache, bilingual output, and path sanitization.
func translateAndWriteHTML(rc io.Reader, writer *zip.Writer, outName, originalName string, p *provider.Provider, apiURL, systemPrompt string, s *settings.Settings, cacheDir string, nameMap map[string]string) error {
	doc, err := goquery.NewDocumentFromReader(rc)
	if err != nil {
		return fmt.Errorf("failed to parse HTML %s: %w", originalName, err)
	}

	// Fix internal links to sanitized file names
	if len(nameMap) > 0 {
		fixInternalLinks(doc, nameMap)
	}

	body := doc.Find("body")
	if body.Length() == 0 {
		fmt.Println("  ⚠ No <body> tag found, skipping")
		htmlStr, _ := doc.Html()
		fw, _ := writer.Create(outName)
		fw.Write([]byte(htmlStr))
		return nil
	}

	bodyHTML, _ := body.Html()
	totalChars := len(bodyHTML)

	if strings.TrimSpace(bodyHTML) == "" {
		fmt.Println("  ⚠ Empty body, skipping")
		htmlStr, _ := doc.Html()
		fw, _ := writer.Create(outName)
		fw.Write([]byte(htmlStr))
		return nil
	}

	// Split into chunks using settings
	chunks := splitBodyIntoChunks(bodyHTML, s.MaxChunkChars)
	totalChunks := len(chunks)

	fmt.Printf("  📊 Total: %d characters → %d chunks\n", totalChars, totalChunks)
	fmt.Printf("  🚀 Translating (max %d parallel)...\n", s.MaxConcurrent)

	// Translate all chunks concurrently with cache support
	translatedChunks := make([]string, totalChunks)
	originalChunks := make([]string, totalChunks)
	copy(originalChunks, chunks)

	var mu sync.Mutex
	var wg sync.WaitGroup
	completed := 0

	sem := make(chan struct{}, s.MaxConcurrent)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, chunkHTML string) {
			defer wg.Done()

			// Check cache first
			cached := cache.LoadChunk(cacheDir, originalName, idx)
			if cached != "" {
				mu.Lock()
				translatedChunks[idx] = cached
				completed++
				fmt.Printf("  ⏩ Chunk %d/%d loaded from cache\n", completed, totalChunks)
				mu.Unlock()
				return
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			chunkLen := len(chunkHTML)
			translated := translator.Translate(p, apiURL, systemPrompt, chunkHTML)

			// Save to cache
			cache.SaveChunk(cacheDir, originalName, idx, translated)

			mu.Lock()
			translatedChunks[idx] = translated
			completed++
			fmt.Printf("  ✓ Chunk %d/%d done (%d chars)\n", completed, totalChunks, chunkLen)
			mu.Unlock()
		}(i, chunk)
	}

	wg.Wait()

	// Build final body
	var finalBody string
	if s.Bilingual {
		// Bilingual: original chunk + translated chunk with styling
		for i := range chunks {
			finalBody += originalChunks[i]
			finalBody += `<div class="babelbook-translated" style="color:#555; border-left:3px solid #4a90d9; padding-left:10px; margin:8px 0; font-style:italic;">`
			finalBody += translatedChunks[i]
			finalBody += `</div>`
		}
	} else {
		finalBody = strings.Join(translatedChunks, "")
	}

	body.SetHtml(finalBody)

	fmt.Printf("  ✅ File complete: %d chunks translated\n", totalChunks)

	htmlStr, _ := doc.Html()
	fileWriter, err := writer.Create(outName)
	if err != nil {
		return fmt.Errorf("failed to create entry %s: %w", outName, err)
	}
	fileWriter.Write([]byte(htmlStr))
	return nil
}
