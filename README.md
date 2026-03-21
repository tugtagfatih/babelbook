# Babelbook: EPUB Translation Engine

Babelbook is a fast, concurrent, multi-provider AI-powered CLI tool that translates EPUB books into any language while preserving HTML formatting.

Supports **Gemini**, **OpenAI**, and **Anthropic** out of the box.

## Features

- 🤖 **Multi-Provider AI** — Gemini, OpenAI, Anthropic with one-click switching
- 📖 **Bilingual Output** — Dual-language EPUBs for language learning
- 💾 **Resume Capability** — Interrupted translations pick up where they left off
- 📊 **Progress Bar** — Visual progress with ETA across all chunks
- 📚 **Batch Translation** — Translate all EPUBs in a folder at once
- 📖 **Glossary** — Define terms for consistent translations (e.g. character names)
- 💰 **Cost Estimation** — See estimated tokens and cost before translating
- ⚙️ **TUI Settings** — Adjust chunk size, parallelism, bilingual, extra prompt
- 🖥️ **CLI Flags** — Non-interactive mode for scripts and automation
- ✏️ **Custom Prompts** — "Use formal tone", "Don't translate names", etc.
- 🔧 **Path Sanitization** — Handles unicode/special chars in EPUB filenames
- 🛡️ **Resilient** — Auto-retry with detailed error reporting

---

## Quick Start

```bash
git clone https://github.com/tugtagfatih/babelbook.git
cd babelbook
cp .env.example .env   # Add your API key
go run .
```

Pre-built binaries: [Releases](https://github.com/tugtagfatih/babelbook/releases)

---

## Configuration

Add at least one API key to `.env`:
```env
GEMINI_API_KEY=your_key_here
OPENAI_API_KEY=your_key_here
ANTHROPIC_API_KEY=your_key_here
```

---

## Usage

### Interactive Mode (TUI)
```bash
go run .
# or
./babelbook
```

1. Provider & model auto-loaded (or selected on first run)
2. **Main Menu**: Start translation, Settings, or Change model
3. Select file (or **[A] Translate ALL** for batch)
4. Choose languages → see cost estimate → translate with progress bar

### CLI Mode (Automation)
```bash
# Single file
babelbook --input "book.epub" --target "Turkish"

# Batch (all EPUBs in current dir)
babelbook --batch --target "Turkish"

# With options
babelbook --input "book.epub" --target "Turkish" --bilingual --glossary "glossary.txt"
```

| Flag | Description |
|------|-------------|
| `--input` | Input EPUB file |
| `--target` | Target language (e.g. Turkish) |
| `--source` | Source language (default: English) |
| `--bilingual` | Generate dual-language EPUB |
| `--batch` | Translate all EPUBs in current directory |
| `--glossary` | Path to glossary file |

### Settings Menu
```
⚙ Current Settings:
  [1] Chunk size      : 50000 chars
  [2] Max parallel    : 20 requests
  [3] Bilingual mode  : OFF
  [4] Extra prompt    : (none)
  [0] ← Back
```

All settings are automatically saved across sessions.

---

## Glossary

Create a `glossary.txt` in the same directory (auto-detected) or pass with `--glossary`:

```
# Character names — don't translate
John -> John
Hogwarts -> Hogwarts

# Custom translations
Wand -> Asa
Stormlight -> Fırtınaışığı
```

Format: `source -> target` (one per line, `#` for comments)

---

## Cost Estimation

Before translation starts, Babelbook shows:
```
💰 Estimate: ~125k tokens, 12 chunks (free tier / ~$0.0125)
```

| Provider | Input $/1M | Output $/1M |
|----------|-----------|-------------|
| Gemini   | $0.10     | $0.40       |
| OpenAI   | $2.50     | $10.00      |
| Anthropic| $3.00     | $15.00      |

---

## Rate Limits

Defaults optimized for: **1K RPM · 2M TPM · 10K RPD**

Adjust via Settings menu:
- **Higher RPM**: Increase max parallel
- **Lower RPM**: Decrease max parallel
- **Larger output limit**: Increase chunk size

---

## Supported Providers

| Provider | Default Model | Others |
|----------|---------------|--------|
| Gemini | `gemini-3-flash-preview` | gemini-2.5-flash, gemini-2.5-pro, gemini-2.0-flash |
| OpenAI | `gpt-4o` | gpt-4o-mini, o3-mini |
| Anthropic | `claude-sonnet-4-20250514` | claude-3-5-haiku |

---

## Project Structure

```
babelbook/
├── main.go              # Entry + CLI flags
├── settings/            # Persistent settings
├── provider/            # AI providers
├── translator/          # Translation + cost estimation
├── epub/                # EPUB I/O + chunking
├── cache/               # Resume capability
├── glossary/            # Term mapping
├── progress/            # Visual progress bar
├── config/              # .env loading
├── ui/                  # Interactive CLI
└── .github/workflows/   # CI/CD
```

## License

MIT License
