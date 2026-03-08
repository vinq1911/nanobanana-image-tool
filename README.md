# nanobanana-image-tool

A standalone image generation tool powered by the Nano Banana 2 model. Designed for AI agent workflows — submit a prompt, get an illustration back.

Primary use case: generating children's book illustrations for [bookgen-service](https://github.com/vinq1911/bookgen-service).

## Features

- **MCP server** — plug directly into Claude Code / Claude Desktop as a tool
- **Two inference providers**: Google Gemini API (canonical) and fal.ai
- CLI and HTTP API interfaces
- Deterministic output with seed control
- Structured JSON metadata for every generated image
- Go client library for easy integration

## Quick Start

```bash
# Set your API key in .env (or export it)
echo 'GOOGLE_API_KEY=your-key' > .env

# Build both binaries (CLI + MCP server)
make build

# Generate an image via CLI
./bin/nanobanana-tool generate \
  --prompt "friendly robot in a colorful town, children's book illustration" \
  --width 1024 \
  --height 1024 \
  --output ./output

# Start the HTTP server
make serve
```

## MCP Server (Claude integration)

The MCP server exposes `generate_illustration` as a tool that Claude can call directly.

### Build

```bash
make build          # builds both nanobanana-tool and nanobanana-mcp
# or just the MCP server:
make build-mcp
```

### Configure in Claude Code

Add to `~/.claude/claude_code_config.json`:

```json
{
  "mcpServers": {
    "nanobanana": {
      "command": "/absolute/path/to/bin/nanobanana-mcp",
      "env": {
        "GOOGLE_API_KEY": "your-gemini-api-key"
      }
    }
  }
}
```

Or if you use fal.ai:

```json
{
  "mcpServers": {
    "nanobanana": {
      "command": "/absolute/path/to/bin/nanobanana-mcp",
      "env": {
        "NANOBANANA_PROVIDER": "falai",
        "FAL_KEY": "your-fal-key"
      }
    }
  }
}
```

### Configure in Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "nanobanana": {
      "command": "/absolute/path/to/bin/nanobanana-mcp",
      "env": {
        "GOOGLE_API_KEY": "your-gemini-api-key"
      }
    }
  }
}
```

Once configured, Claude sees the `generate_illustration` tool and can call it directly. The tool returns the generated image as base64 (visible inline) plus JSON metadata with the saved file path.

### Tool: generate_illustration

| Parameter | Type | Required | Description |
|---|---|---|---|
| `prompt` | string | yes | Illustration description |
| `negative_prompt` | string | no | Things to avoid |
| `width` | number | no | Width in pixels (default: 1024) |
| `height` | number | no | Height in pixels (default: 1024) |
| `seed` | number | no | Random seed for reproducibility |
| `style` | string | no | Visual style (watercolor, crayon, etc.) |
| `output_dir` | string | no | Save directory (default: ./output) |
| `image_format` | string | no | png or jpg (default: png) |

## Providers

### Google Gemini API (default)

Uses the official Google Gen AI Go SDK. This is the canonical Nano Banana 2 backend.

| Variable | Default | Description |
|---|---|---|
| `NANOBANANA_PROVIDER` | `gemini` | Set to `gemini` |
| `NANOBANANA_GEMINI_API_KEY` | | API key (also reads `GEMINI_API_KEY`, `GOOGLE_API_KEY`) |
| `NANOBANANA_GEMINI_MODEL` | `gemini-2.5-flash-image` | Model ID |

Get a key at [Google AI Studio](https://aistudio.google.com/).

### fal.ai

Uses the fal.ai REST API with queue-based async inference.

| Variable | Default | Description |
|---|---|---|
| `NANOBANANA_PROVIDER` | | Set to `falai` |
| `NANOBANANA_FALAI_KEY` | | API key (also reads `FAL_KEY`) |
| `NANOBANANA_FALAI_MODEL` | `fal-ai/nano-banana-2` | Model ID |

Get a key at [fal.ai](https://fal.ai/).

## CLI Usage

### generate

```bash
nanobanana-tool generate [flags]

Flags:
  --prompt           Image prompt (required)
  --negative-prompt  Negative prompt
  --width            Image width (default: 1024)
  --height           Image height (default: 1024)
  --seed             Random seed, -1 for random (default: -1)
  --style            Style preset
  --output           Output directory (default: ./output)
  --format           Image format: png or jpg (default: png)
```

Output (JSON to stdout):

```json
{
  "image_path": "/absolute/path/to/image.png",
  "width": 1024,
  "height": 1024,
  "seed": 12345,
  "prompt": "friendly robot in a colorful town",
  "model": "nanobanana-2"
}
```

### serve

```bash
nanobanana-tool serve
```

Starts the HTTP API server.

## HTTP API

### POST /generate

```bash
curl -X POST http://localhost:8080/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "friendly robot in a colorful town",
    "width": 1024,
    "height": 1024,
    "seed": 42,
    "image_format": "png"
  }'
```

Response:

```json
{
  "image_path": "/absolute/path/to/image.png",
  "metadata": {
    "image_path": "/absolute/path/to/image.png",
    "width": 1024,
    "height": 1024,
    "seed": 42,
    "prompt": "friendly robot in a colorful town",
    "model": "nanobanana-2"
  }
}
```

### GET /health

Returns `{"status": "ok"}`.

### GET /tool-schema

Returns the Anthropic tool use JSON schema for `generate_illustration`.

## Configuration

All configuration is via environment variables or a `.env` file in the working directory. See `configs/default.env` for a full reference.

| Variable | Default | Description |
|---|---|---|
| `NANOBANANA_PROVIDER` | `gemini` | `gemini` or `falai` |
| `NANOBANANA_OUTPUT_DIR` | `./output` | Default output directory |
| `NANOBANANA_PORT` | `8080` | HTTP server port |
| `NANOBANANA_LOG_LEVEL` | `info` | Log level (`info` or `debug`) |

## Integration with bookgen-service

Use the Go client from `pkg/client`:

```go
import (
    "github.com/vinq1911/nanobanana-image-tool/internal/models"
    "github.com/vinq1911/nanobanana-image-tool/pkg/client"
)

nb := client.New("http://localhost:8080")

seed := int64(42)
resp, err := nb.Generate(ctx, models.GenerateRequest{
    Prompt:      "a cat reading a book under a tree",
    Width:       1024,
    Height:      1024,
    Seed:        &seed,
    ImageFormat: "png",
})
// resp.ImagePath contains the generated image path
```

See `examples/bookgen_integration.go` for a full working example.

## Project Structure

```
nanobanana-image-tool/
  cmd/
    nanobanana-tool/        # CLI + HTTP server entrypoint
    nanobanana-mcp/         # MCP server entrypoint (for Claude)
  internal/
    api/                    # HTTP server + tool schema
    config/                 # Environment-driven config with .env loading
    generator/
      generator.go          # ImageGenerator interface
      gemini.go             # Google Gemini API provider
      falai.go              # fal.ai provider
      nanobanana.go         # Provider factory
    models/                 # Request/response types
    storage/                # Image persistence
    logging/                # Structured logger
  pkg/client/               # Go client library for integration
  configs/                  # Example configuration
  examples/                 # Integration examples
  tool_schema.json          # Anthropic tool use schema (standalone)
```

## Development

```bash
make build        # Build both binaries (CLI + MCP)
make build-mcp    # Build MCP server only
make test         # Run tests
make serve        # Build and start HTTP server
make clean        # Remove build artifacts and output
```
