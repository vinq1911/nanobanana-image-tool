package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vinq1911/nanobanana-image-tool/internal/config"
	"github.com/vinq1911/nanobanana-image-tool/internal/generator"
	"github.com/vinq1911/nanobanana-image-tool/internal/logging"
	"github.com/vinq1911/nanobanana-image-tool/internal/models"
	"github.com/vinq1911/nanobanana-image-tool/internal/storage"
)

func main() {
	logger := logging.New()
	cfg := config.Load()

	gen, err := generator.New(cfg, logger)
	if err != nil {
		log.Fatalf("failed to create generator: %v", err)
	}

	store := storage.NewLocalStorage(logger)

	// Create MCP server.
	s := server.NewMCPServer(
		"nanobanana-image-tool",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Define the generate_illustration tool.
	tool := mcp.NewTool("generate_illustration",
		mcp.WithDescription(
			"Generate a children's book illustration using the Nano Banana 2 image model. "+
				"Accepts a text prompt and returns the generated image (base64) plus metadata. "+
				"The image is also saved to disk.",
		),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("A detailed description of the illustration to generate. Be specific about subjects, setting, mood, and visual style."),
		),
		mcp.WithString("negative_prompt",
			mcp.Description("Things to avoid in the generated image. Example: 'scary, dark, violent'"),
		),
		mcp.WithNumber("width",
			mcp.Description("Image width in pixels. Defaults to 1024."),
		),
		mcp.WithNumber("height",
			mcp.Description("Image height in pixels. Defaults to 1024."),
		),
		mcp.WithNumber("seed",
			mcp.Description("Random seed for deterministic generation. Omit for random."),
		),
		mcp.WithString("style",
			mcp.Description("Visual style preset. Examples: 'watercolor', 'crayon', 'digital illustration', 'pencil sketch'."),
		),
		mcp.WithString("output_dir",
			mcp.Description("Directory to save the generated image. Defaults to ./output."),
		),
		mcp.WithString("image_format",
			mcp.Description("Output format: 'png' or 'jpg'. Defaults to 'png'."),
			mcp.Enum("png", "jpg"),
		),
	)

	// Register tool handler.
	s.AddTool(tool, makeHandler(cfg, gen, store))

	// Run as stdio server.
	stdio := server.NewStdioServer(s)
	if err := stdio.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatalf("mcp server error: %v", err)
	}
}

func makeHandler(
	cfg *config.Config,
	gen generator.ImageGenerator,
	store storage.Storage,
) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req := models.GenerateRequest{
			Prompt:         request.GetString("prompt", ""),
			NegativePrompt: request.GetString("negative_prompt", ""),
			Width:          request.GetInt("width", 1024),
			Height:         request.GetInt("height", 1024),
			Style:          request.GetString("style", ""),
			OutputDir:      request.GetString("output_dir", cfg.OutputDir),
			ImageFormat:    request.GetString("image_format", "png"),
		}

		if seedVal := request.GetFloat("seed", -1); seedVal >= 0 {
			s := int64(seedVal)
			req.Seed = &s
		}

		if err := req.Validate(); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, imgData, err := gen.Generate(ctx, req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("generation failed: %v", err)), nil
		}

		imgPath, err := store.Save(ctx, imgData, req.ImageFormat, req.OutputDir)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to save image: %v", err)), nil
		}

		result.ImagePath = imgPath

		// Build metadata JSON.
		metadata, _ := json.Marshal(result)

		// Return image + metadata text.
		mimeType := "image/png"
		if req.ImageFormat == "jpg" {
			mimeType = "image/jpeg"
		}
		b64 := base64.StdEncoding.EncodeToString(imgData)

		return mcp.NewToolResultImage(string(metadata), b64, mimeType), nil
	}
}
