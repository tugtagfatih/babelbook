# Babelbook: EPUB Translation Engine

Babelbook is a fast, concurrent, multi-provider AI-powered CLI tool designed to translate EPUB books into any language while preserving their original HTML/XHTML formatting and chapter structures.

It supports **Gemini**, **OpenAI**, and **Anthropic (Claude)** models out of the box.

## Features

- **Multi-Provider AI Support**: Switch between Gemini, OpenAI, and Anthropic.
- **Bilingual Output**: Generate dual-language EPUBs with original text above and translation below — perfect for language learning.
- **Resume Capability**: If translation is interrupted (network error, API limit), restart and it picks up exactly where it left off.
- **Concurrent Translation**: Splits content into chunks and translates them all in parallel.
- **TUI Settings Menu**: Adjust chunk size, parallelism, bilingual mode, and extra prompts without editing code.
- **Custom Prompts**: Add translation instructions like "use formal tone" or "don't translate character names".
- **Path Sanitization**: Handles EPUBs with unicode or special characters in filenames.
- **Format Preservation**: Translates only text content — HTML structure, CSS, and images remain untouched.
- **Resilience**: Built-in retry mechanism with detailed error reporting.

---

## Installation

```bash
git clone https://github.com/tugtagfatih/babelbook.git
cd babelbook
go build -o babelbook .
```

Pre-built binaries: [Releases](https://github.com/tugtagfatih/babelbook/releases)

---

## Configuration

1. Copy the template: `cp .env.example .env`
2. Add at least one API key:
   ```env
   GEMINI_API_KEY=your_key_here
   OPENAI_API_KEY=your_key_here
   ANTHROPIC_API_KEY=your_key_here
   ```

---

## Usage

```bash
go run .
# or
./babelbook
```

### Interactive Flow
1. Select AI provider (auto-selected if only one key exists)
2. Select model (default: `gemini-3-flash-preview`)
3. **Main Menu**: Start translation or open Settings
4. Select EPUB file and languages
5. Translation begins with real-time progress

### Settings Menu
Accessible from the main menu before starting translation:

```
⚙ Current Settings:
  [1] Chunk size      : 50000 chars
  [2] Max parallel    : 20 requests
  [3] Bilingual mode  : OFF
  [4] Extra prompt    : (none)
  [0] ← Back
```

| Setting | Default | Description |
|---------|---------|-------------|
| Chunk size | 50,000 | Characters per API call. Decrease if translations get truncated |
| Max parallel | 20 | Concurrent API requests. Increase for higher RPM tiers |
| Bilingual | OFF | When ON, outputs both original and translated text |
| Extra prompt | (none) | Custom instructions for the translator |

---

## Rate Limits

Defaults are optimized for: **1K RPM · 2M TPM · 10K RPD**

Adjust via the Settings menu:
- **Higher RPM** (e.g. 10K): Set max parallel to `100`
- **Lower RPM** (e.g. 15 free tier): Set max parallel to `2`
- **Larger output limit**: Increase chunk size to `100000`

---

## Resume (Cache)

If translation is interrupted, simply run Babelbook again with the same file. It will:
1. Detect the `.babelbook_cache/` directory
2. Load previously translated chunks
3. Only translate the remaining chunks
4. Clean up cache after successful completion

---

## Supported Providers

| Provider | Default Model | Other Models |
|----------|---------------|--------------|
| Gemini | `gemini-3-flash-preview` | gemini-2.5-flash, gemini-2.5-pro, gemini-2.0-flash |
| OpenAI | `gpt-4o` | gpt-4o-mini, o3-mini |
| Anthropic | `claude-sonnet-4-20250514` | claude-3-5-haiku |

---

## Project Structure

```
babelbook/
├── main.go              # Entry point
├── settings/            # Runtime configuration struct
├── provider/            # AI provider definitions
├── translator/          # Translation logic with retries
├── epub/                # EPUB reading, chunking, writing
├── cache/               # Resume capability (disk cache)
├── config/              # .env loading
├── ui/                  # Interactive CLI + settings menu
└── .github/workflows/   # CI/CD release pipeline
```

## License

MIT License
