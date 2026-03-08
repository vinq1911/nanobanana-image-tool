package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vinq1911/nanobanana-image-tool/internal/models"
	"github.com/vinq1911/nanobanana-image-tool/pkg/client"
)

// This example shows how bookgen-service integrates with nanobanana-image-tool.
//
// Workflow:
//   1. bookgen-service generates an illustration prompt for a page.
//   2. It calls nanobanana-image-tool via the client.
//   3. It receives the image path and attaches it to the page layout.
func main() {
	// Point to the running nanobanana-image-tool HTTP server.
	nb := client.New("http://localhost:8080")

	ctx := context.Background()

	// Check health first.
	if err := nb.Health(ctx); err != nil {
		log.Fatalf("nanobanana service is not healthy: %v", err)
	}

	// Generate an illustration for a children's book page.
	seed := int64(42)
	resp, err := nb.Generate(ctx, models.GenerateRequest{
		Prompt:         "a friendly robot waving hello in a colorful town, soft watercolor style, children's book illustration",
		NegativePrompt: "scary, dark, violent",
		Width:          1024,
		Height:         1024,
		Seed:           &seed,
		Style:          "watercolor",
		OutputDir:      "./output/bookgen",
		ImageFormat:    "png",
	})
	if err != nil {
		log.Fatalf("generation failed: %v", err)
	}

	fmt.Printf("Image generated!\n")
	fmt.Printf("  Path:  %s\n", resp.ImagePath)
	fmt.Printf("  Model: %s\n", resp.Metadata.Model)
	fmt.Printf("  Seed:  %d\n", resp.Metadata.Seed)
	fmt.Printf("  Size:  %dx%d\n", resp.Metadata.Width, resp.Metadata.Height)
}
