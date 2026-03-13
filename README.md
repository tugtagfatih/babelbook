# Babelbook: EPUB Translation Engine

Babelbook is a fast, interactive, and multi-provider AI-powered CLI tool designed to translate EPUB books into any language while preserving exactly their original HTML/XHTML formatting and chapter structures.

It currently supports **Gemini**, **OpenAI**, and **Anthropic (Claude)** models out of the box.

## Features

- **Multi-Provider AI Support**: Switch effortlessly between Gemini, OpenAI (ChatGPT), and Anthropic (Claude).
- **Interactive CLI**: Easy-to-use terminal interface for selecting files, providers, models, and languages.
- **Format Preservation**: Parses the internal HTML structure of EPUB files, translating only the text (`<p>`, `<h1>`, `<span>`, etc.) without breaking styling or layouts.
- **Real-Time Progress**: Detailed, element-by-element progress tracking, showing exactly what is being translated and how many elements remain.
- **Resilience**: Built-in exponential backoff and retry mechanisms for API failures.
- **Custom Models**: Plug in any provider-compatible endpoint via the custom model entry flow.

---

## Installation

Ensure you have Go installed on your system.

```bash
# Clone the repository
git clone https://github.com/tugtagfatih/babelbook.git
cd babelbook

# Build the executable
go build ./...

# Or run directly
go run .
```

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
*Note: You only need to provide the key for the AI provider you intend to use. If multiple keys are provided, Babelbook will ask you to choose one at startup.*

---

## Usage

1. Place the `.epub` files you wish to translate into the root directory of the project.
2. Run the application:
   ```bash
   go run .
   ```
3. Follow the interactive prompts to:
   - Select your AI Provider.
   - Select the specific model (e.g., `gemini-2.5-flash`, `gpt-4o`).
   - Select the target `.epub` file from the list.
   - Enter your source and target languages.

The translation will begin immediately. A new file will be generated with a language prefix (e.g., `TU_book.epub` for Turkish).

---

## Rate Limiting & Speed Customization

To prevent API bans (HTTP 429 errors), Babelbook implements a default delay between consecutive API requests.

**Default Speed**: 1 Request per second (~60 Requests Per Minute / RPM).

### How to change the speed
If you have a paid API tier with higher limits (or want to slow it down for free tiers), you can modify this easily in the code:

1. Open `epub/epub.go`
2. Locate line 18:
   ```go
   // rateLimitDelay is the pause between API calls to avoid rate limiting.
   const rateLimitDelay = 1 * time.Second
   ```
3. Change it to your desired speed.
   - **Faster** (e.g., 200 RPM): `const rateLimitDelay = 300 * time.Millisecond`
   - **Maximum Speed** (No limit): `const rateLimitDelay = 0 * time.Second`
   - **Slower** (e.g., 15 RPM for free tiers): `const rateLimitDelay = 4 * time.Second`
4. Rebuild the application (`go build .`).

---

## Project Structure

- `main.go`: Application entry point and orchestrator.
- `provider/`: Definitions and request/response parsers for Gemini, OpenAI, and Anthropic.
- `translator/`: The core translation loop logic handling retries and errors.
- `epub/`: Handles reading ZIP/EPUB archives, parsing HTML via GoQuery, and recompiling the book.
- `config/`: Environment loading and validation.
- `ui/`: All interactive command-line interface flows.

---

## License

MIT License
