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
	"github.com/tugtagfatih/babelbook/progress"
	"github.com/tugtagfatih/babelbook/provider"
	"github.com/tugtagfatih/babelbook/settings"
	"github.com/tugtagfatih/babelbook/translator"
)

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
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
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

// AnalyzeEPUB scans an EPUB file and returns total character count and total HTML chunks.
func AnalyzeEPUB(inputPath string, maxChunkChars int) (totalChars int, totalChunks int, err error) {
	reader, err := zip.OpenReader(inputPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if !isHTMLFile(file.Name) {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			continue
		}
		doc, err := goquery.NewDocumentFromReader(rc)
		rc.Close()
		if err != nil {
			continue
		}
		body := doc.Find("body")
		if body.Length() == 0 {
			continue
		}
		html, _ := body.Html()
		trimmed := strings.TrimSpace(html)
		if trimmed == "" {
			continue
		}
		totalChars += len(trimmed)
		chunks := splitBodyIntoChunks(trimmed, maxChunkChars)
		totalChunks += len(chunks)
	}
	return totalChars, totalChunks, nil
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

	// Build name sanitization mapping
	nameMap := make(map[string]string)
	for _, file := range reader.File {
		sanitized := sanitizePath(file.Name)
		if sanitized != file.Name {
			nameMap[file.Name] = sanitized
		}
	}

	// Count total chunks for global progress bar
	totalChunksGlobal := 0
	for _, file := range reader.File {
		if !isHTMLFile(file.Name) {
			continue
		}
		rc, _ := file.Open()
		if rc == nil {
			continue
		}
		doc, err := goquery.NewDocumentFromReader(rc)
		rc.Close()
		if err != nil {
			continue
		}
		body := doc.Find("body")
		if body.Length() == 0 {
			continue
		}
		html, _ := body.Html()
		if strings.TrimSpace(html) == "" {
			continue
		}
		chunks := splitBodyIntoChunks(html, s.MaxChunkChars)
		totalChunksGlobal += len(chunks)
	}

	globalBar := progress.New(totalChunksGlobal, "📖 Total")
	totalHTML := countHTMLFiles(reader.File)
	processedHTML := 0

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in EPUB: %w", file.Name, err)
		}

		outName := file.Name
		if sanitized, ok := nameMap[file.Name]; ok {
			outName = sanitized
		}

		if isHTMLFile(file.Name) {
			processedHTML++
			fmt.Printf("\n📄 [%d/%d] %s\n", processedHTML, totalHTML, file.Name)

			if err := translateAndWriteHTML(rc, writer, outName, file.Name, p, apiURL, systemPrompt, s, cacheDir, nameMap, globalBar); err != nil {
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

	globalBar.Done()

	cache.Clean(cacheDir)
	fmt.Println("🗑  Cache cleaned.")

	return nil
}

// fixInternalLinks updates src/href attributes in HTML to match sanitized filenames.
func fixInternalLinks(doc *goquery.Document, nameMap map[string]string) {
	for original, sanitized := range nameMap {
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

// translateAndWriteHTML translates HTML content with progress bar, cache, and bilingual support.
func translateAndWriteHTML(rc io.Reader, writer *zip.Writer, outName, originalName string, p *provider.Provider, apiURL, systemPrompt string, s *settings.Settings, cacheDir string, nameMap map[string]string, globalBar *progress.Bar) error {
	doc, err := goquery.NewDocumentFromReader(rc)
	if err != nil {
		return fmt.Errorf("failed to parse HTML %s: %w", originalName, err)
	}

	if len(nameMap) > 0 {
		fixInternalLinks(doc, nameMap)
	}

	body := doc.Find("body")
	if body.Length() == 0 {
		htmlStr, _ := doc.Html()
		fw, _ := writer.Create(outName)
		fw.Write([]byte(htmlStr))
		return nil
	}

	bodyHTML, _ := body.Html()

	if strings.TrimSpace(bodyHTML) == "" {
		htmlStr, _ := doc.Html()
		fw, _ := writer.Create(outName)
		fw.Write([]byte(htmlStr))
		return nil
	}

	chunks := splitBodyIntoChunks(bodyHTML, s.MaxChunkChars)
	totalChunks := len(chunks)

	// Translate all chunks concurrently with cache support
	translatedChunks := make([]string, totalChunks)
	originalChunks := make([]string, totalChunks)
	copy(originalChunks, chunks)

	var wg sync.WaitGroup
	sem := make(chan struct{}, s.MaxConcurrent)

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, chunkHTML string) {
			defer wg.Done()

			// Check cache first
			cached := cache.LoadChunk(cacheDir, originalName, idx)
			if cached != "" {
				translatedChunks[idx] = cached
				globalBar.Increment()
				return
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			translated := translator.Translate(p, apiURL, systemPrompt, chunkHTML)

			cache.SaveChunk(cacheDir, originalName, idx, translated)
			translatedChunks[idx] = translated
			globalBar.Increment()
		}(i, chunk)
	}

	wg.Wait()

	// Build final body
	var finalBody string
	if s.Bilingual {
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

	htmlStr, _ := doc.Html()
	fileWriter, err := writer.Create(outName)
	if err != nil {
		return fmt.Errorf("failed to create entry %s: %w", outName, err)
	}
	fileWriter.Write([]byte(htmlStr))
	return nil
}
