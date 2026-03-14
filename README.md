# Babelbook: EPUB Translation Engine

Babelbook is a fast, concurrent, multi-provider AI-powered CLI tool designed to translate EPUB books into any language while preserving their original HTML/XHTML formatting and chapter structures.

It supports **Gemini**, **OpenAI**, and **Anthropic (Claude)** models out of the box.

## Features

- **Multi-Provider AI Support**: Switch effortlessly between Gemini, OpenAI (ChatGPT), and Anthropic (Claude).
- **Concurrent Translation**: Splits HTML content into chapter-sized chunks and translates them all in parallel for maximum speed.
- **Format Preservation**: Preserves the internal HTML structure of EPUB files — translates only text content without breaking styling or layouts.
- **Interactive CLI**: Easy-to-use terminal interface for selecting files, providers, models, and languages.
- **Real-Time Progress**: Chunk-by-chunk progress tracking.
- **Resilience**: Built-in retry mechanism with detailed error reporting.
- **Custom Models**: Plug in any provider-compatible model via the custom model entry flow.

---

## Installation

Ensure you have [Go](https://go.dev/dl/) installed on your system.

```bash
# Clone the repository
git clone https://github.com/tugtagfatih/babelbook.git
cd babelbook

# Build the executable
go build -o babelbook .

# Or run directly
go run .
```

Pre-built binaries for Windows, Linux, and macOS are available on the [Releases](https://github.com/tugtagfatih/babelbook/releases) page.

---

## Configuration

Babelbook relies on environment variables for API authentication.

1. Copy the example configuration file:
   ```bash
   cp .env.example .env
   ```
2. Open `.env` and fill in one or more API keys:
   ```env
   GEMINI_API_KEY=your_gemini_key_here
   OPENAI_API_KEY=your_openai_key_here
   ANTHROPIC_API_KEY=your_anthropic_key_here
   ```
> **Note:** You only need to provide the key for the provider you intend to use. If multiple keys are present, Babelbook will ask you to choose one at startup.

---

## Usage

1. Place the `.epub` files you wish to translate into the project directory.
2. Run the application:
   ```bash
   go run .
   ```
3. Follow the interactive prompts:
   - Select your AI Provider (auto-selected if only one key is set).
   - Select the model (default: `gemini-3-flash-preview`).
   - Select the target `.epub` file.
   - Enter your source and target languages.

The translation begins immediately. A new file is generated with a language prefix (e.g., `TU_book.epub` for Turkish).

---

## Rate Limits & Performance Tuning

The default configuration is optimized for the following API limits:

| Parameter | Default Value | Description |
|-----------|--------------|-------------|
| **RPM** (Requests Per Minute) | 1,000 | Max API calls per minute |
| **TPM** (Tokens Per Minute) | 2,000,000 | Max tokens processed per minute |
| **RPD** (Requests Per Day) | 10,000 | Max API calls per day |
| **maxChunkChars** | 50,000 | Characters per translation chunk |
| **maxConcurrent** | 20 | Parallel API requests at once |

### How to customize

If you know your API rate limits differ from the defaults, you can change them in [`epub/epub.go`](epub/epub.go):

```go
// Current defaults are optimized for:
//   RPM (Requests Per Minute) : 1,000
//   TPM (Tokens Per Minute)   : 2,000,000
//   RPD (Requests Per Day)    : 10,000

const maxChunkChars = 50000   // Characters per chunk
const maxConcurrent = 20      // Parallel requests
```

**Examples:**
- **Higher RPM (e.g. 10K RPM)**: Increase `maxConcurrent` to `100`
- **Lower RPM (e.g. 15 RPM free tier)**: Decrease `maxConcurrent` to `2`
- **Larger output token limit**: Increase `maxChunkChars` to `100000`
- **Getting truncated translations**: Decrease `maxChunkChars` to `25000`

After changing, rebuild: `go build -o babelbook .`

---

## Supported Providers & Default Models

| Provider | Env Key | Default Model | Other Models |
|----------|---------|---------------|--------------|
| Gemini | `GEMINI_API_KEY` | `gemini-3-flash-preview` | gemini-2.5-flash, gemini-2.5-pro, gemini-2.0-flash |
| OpenAI | `OPENAI_API_KEY` | `gpt-4o` | gpt-4o-mini, o3-mini |
| Anthropic | `ANTHROPIC_API_KEY` | `claude-sonnet-4-20250514` | claude-3-5-haiku |

You can also enter any custom model name via the "Custom" option in the model selection menu.

---

## Project Structure

```
babelbook/
├── main.go              # Entry point and orchestrator
├── provider/            # AI provider definitions (Gemini, OpenAI, Anthropic)
├── translator/          # Core translation logic with retries
├── epub/                # EPUB reading, chunking, and writing
├── config/              # Environment loading and validation
├── ui/                  # Interactive CLI flows
├── .env.example         # Template configuration file
├── .github/workflows/   # CI/CD release pipeline
└── README.md
```

---

## License

MIT License
