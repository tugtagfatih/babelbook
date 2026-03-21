# Babelbook: EPUB Translation Engine

Babelbook is a fast, concurrent, multi-provider AI-powered CLI tool that translates EPUB books into any language while preserving HTML formatting.

Supports **Gemini**, **OpenAI**, **Anthropic**, and **Local AI** (Ollama, LMStudio, any OpenAI-compatible server).

## Features

- 🤖 **Multi-Provider** — Gemini, OpenAI, Anthropic, Local AI (Ollama/LMStudio)
- 🦙 **Local AI** — Translate offline with your own GPU — no API key, no limits, free
- 📖 **Bilingual Output** — Dual-language EPUBs for language learning
- 🎯 **Partial Translation** — Select specific chapters to translate, skip the rest
- 💾 **Resume Capability** — Interrupted translations pick up where they left off
- 📊 **Progress Bar** — Visual progress with ETA across all chunks
- 📚 **Batch Translation** — Translate all EPUBs in a folder at once
- 📖 **Glossary** — Define terms for consistent translations
- 💰 **Cost Estimation** — See estimated tokens and cost before translating
- ⚙️ **TUI Settings** — Chunk size, parallelism, bilingual, extra prompt (auto-saved)
- 🖥️ **CLI Flags** — Non-interactive mode for automation
- 🧠 **Remember Everything** — Provider, model, and languages saved between sessions
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

Add at least one API key or local server URL to `.env`:
```env
GEMINI_API_KEY=your_key_here
OPENAI_API_KEY=your_key_here
ANTHROPIC_API_KEY=your_key_here

# Local AI (Ollama, LMStudio, or any OpenAI-compatible server)
LOCAL_AI_URL=http://localhost:11434    # Ollama
# LOCAL_AI_URL=http://localhost:1234   # LMStudio
```

---

## Local AI Setup

### Ollama
```bash
# Install: https://ollama.com
ollama pull llama3.1
# Add to .env: LOCAL_AI_URL=http://localhost:11434
```

### LMStudio
1. Download from [lmstudio.ai](https://lmstudio.ai)
2. Load a model and start the server
3. Add to `.env`: `LOCAL_AI_URL=http://localhost:1234`

### Any OpenAI-compatible server
Set `LOCAL_AI_URL` to your server's base URL. Babelbook will auto-append `/v1/chat/completions`.

When selecting a Local AI model, choose "Custom" and enter your model name (e.g. `llama3.1`, `mistral`, `gemma`).

---

## Usage

### Interactive Mode
```bash
go run .
```

1. Provider & model auto-loaded (saved from last session)
2. **Main Menu**: Start translation / Settings / Change model
3. Select file (or **[A] Translate ALL** for batch)
4. Choose languages (remembered from last session)
5. **Select chapters** (translate specific ones or all)
6. See cost estimate → translate with progress bar

### CLI Mode
```bash
babelbook --input "book.epub" --target "Turkish"
babelbook --batch --target "Turkish" --bilingual
babelbook --input "book.epub" --target "Turkish" --glossary "glossary.txt"
```

| Flag | Description |
|------|-------------|
| `--input` | Input EPUB file |
| `--target` | Target language |
| `--source` | Source language (default: English) |
| `--bilingual` | Dual-language output |
| `--batch` | Translate all EPUBs |
| `--glossary` | Path to glossary file |

---

## Glossary

Create `glossary.txt` (auto-detected) or pass with `--glossary`:
```
# Don't translate character names
John -> John
Hogwarts -> Hogwarts

# Custom translations
Wand -> Asa
```

---

## Supported Providers

| Provider | Env Variable | Default Model |
|----------|-------------|---------------|
| Gemini | `GEMINI_API_KEY` | gemini-3-flash-preview |
| OpenAI | `OPENAI_API_KEY` | gpt-4o |
| Anthropic | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250514 |
| Local AI | `LOCAL_AI_URL` | auto (enter your model name) |

---

## Project Structure

```
babelbook/
├── main.go              # Entry + CLI flags
├── settings/            # Persistent settings (JSON)
├── provider/            # AI providers (Gemini, OpenAI, Anthropic, Local AI)
├── translator/          # Translation + cost estimation
├── epub/                # EPUB I/O + chunking + chapter listing
├── cache/               # Resume capability
├── glossary/            # Term mapping
├── progress/            # Visual progress bar
├── config/              # .env loading
├── ui/                  # Interactive CLI + settings menu
└── .github/workflows/   # CI/CD release pipeline
```

## License

MIT License
