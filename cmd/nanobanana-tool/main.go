package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vinq1911/nanobanana-image-tool/internal/api"
	"github.com/vinq1911/nanobanana-image-tool/internal/config"
	"github.com/vinq1911/nanobanana-image-tool/internal/generator"
	"github.com/vinq1911/nanobanana-image-tool/internal/logging"
	"github.com/vinq1911/nanobanana-image-tool/internal/models"
	"github.com/vinq1911/nanobanana-image-tool/internal/storage"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		runGenerate(os.Args[2:])
	case "serve":
		runServe()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`nanobanana-tool — Nano Banana 2 image generation tool

Usage:
  nanobanana-tool <command> [flags]

Commands:
  generate    Generate an image from a prompt
  serve       Start the HTTP API server
  help        Show this help message

Environment:
  NANOBANANA_PROVIDER          gemini (default) or falai
  NANOBANANA_GEMINI_API_KEY    Google Gemini API key (or GEMINI_API_KEY / GOOGLE_API_KEY)
  NANOBANANA_FALAI_KEY         fal.ai API key (or FAL_KEY)

Use "nanobanana-tool <command> --help" for more information.`)
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	prompt := fs.String("prompt", "", "Image generation prompt (required)")
	negativePrompt := fs.String("negative-prompt", "", "Negative prompt")
	width := fs.Int("width", 1024, "Image width in pixels")
	height := fs.Int("height", 1024, "Image height in pixels")
	seed := fs.Int64("seed", -1, "Random seed (-1 for random)")
	style := fs.String("style", "", "Style preset")
	outputDir := fs.String("output", "./output", "Output directory")
	imageFormat := fs.String("format", "png", "Image format (png or jpg)")

	fs.Parse(args)

	if *prompt == "" {
		fmt.Fprintln(os.Stderr, "error: --prompt is required")
		fs.Usage()
		os.Exit(1)
	}

	logger := logging.New()
	cfg := config.Load()

	gen, err := generator.New(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	req := models.GenerateRequest{
		Prompt:         *prompt,
		NegativePrompt: *negativePrompt,
		Width:          *width,
		Height:         *height,
		Style:          *style,
		OutputDir:      *outputDir,
		ImageFormat:    *imageFormat,
	}
	if *seed >= 0 {
		req.Seed = seed
	}

	store := storage.NewLocalStorage(logger)
	ctx := context.Background()

	result, imgData, err := gen.Generate(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	imgPath, err := store.Save(ctx, imgData, req.ImageFormat, *outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error saving image: %v\n", err)
		os.Exit(1)
	}

	result.ImagePath = imgPath

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func runServe() {
	logger := logging.New()
	cfg := config.Load()

	gen, err := generator.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create generator", "error", err)
		os.Exit(1)
	}

	store := storage.NewLocalStorage(logger)
	srv := api.NewServer(cfg, gen, store, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		// Server failed to start or crashed.
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		// Received SIGINT/SIGTERM — shut down gracefully with a timeout.
		logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown error", "error", err)
			os.Exit(1)
		}
		logger.Info("server stopped")
	}
}
