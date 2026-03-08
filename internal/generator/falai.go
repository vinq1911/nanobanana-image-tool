package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vinq1911/nanobanana-image-tool/internal/config"
	"github.com/vinq1911/nanobanana-image-tool/internal/models"
)

const (
	falQueueBaseURL   = "https://queue.fal.run"
	falUploadURL      = "https://fal.ai/api/storage/upload/initiate"
	falPollInterval   = 2 * time.Second
	falMaxWait        = 5 * time.Minute
)

// FalAIGenerator implements ImageGenerator using the fal.ai REST API
// with queue-based async inference.
type FalAIGenerator struct {
	cfg    *config.Config
	logger *slog.Logger
	client *http.Client
}

// NewFalAIGenerator creates a new fal.ai-based image generator.
func NewFalAIGenerator(cfg *config.Config, logger *slog.Logger) *FalAIGenerator {
	return &FalAIGenerator{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// fal.ai queue submit response.
type falQueueResponse struct {
	RequestID   string `json:"request_id"`
	ResponseURL string `json:"response_url"`
	StatusURL   string `json:"status_url"`
}

// fal.ai queue status response.
type falStatusResponse struct {
	Status string `json:"status"`
}

// fal.ai final result response.
type falResultResponse struct {
	Images []struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
	} `json:"images"`
}

func (g *FalAIGenerator) Generate(ctx context.Context, req models.GenerateRequest) (*models.ImageResult, []byte, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}

	seed := resolveSeed(req.Seed)

	g.logger.Info("generating image via fal.ai",
		"model", g.cfg.FalAIModel,
		"prompt", req.Prompt,
		"seed", seed,
	)

	// Upload reference images to fal.ai storage if present.
	var refURLs []string
	for _, ref := range req.ReferenceImages {
		u, err := g.uploadToFalStorage(ctx, ref.Data, ref.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: uploading reference %q: %v", models.ErrGeneration, ref.Name, err)
		}
		refURLs = append(refURLs, u)
	}

	// 1. Submit to queue.
	queueResp, err := g.submitToQueue(ctx, req, seed, refURLs)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: fal.ai submit: %v", models.ErrGeneration, err)
	}

	g.logger.Debug("queued", "request_id", queueResp.RequestID)

	// 2. Poll until complete.
	if err := g.pollUntilDone(ctx, queueResp.StatusURL); err != nil {
		return nil, nil, fmt.Errorf("%w: fal.ai poll: %v", models.ErrGeneration, err)
	}

	// 3. Fetch result.
	falResult, err := g.fetchResult(ctx, queueResp.ResponseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: fal.ai result: %v", models.ErrGeneration, err)
	}

	if len(falResult.Images) == 0 {
		return nil, nil, fmt.Errorf("%w: fal.ai returned no images", models.ErrGeneration)
	}

	// 4. Download the image.
	imgData, err := g.downloadImage(ctx, falResult.Images[0].URL)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: downloading image: %v", models.ErrGeneration, err)
	}

	width := falResult.Images[0].Width
	height := falResult.Images[0].Height
	if width == 0 {
		width = req.Width
	}
	if height == 0 {
		height = req.Height
	}

	result := &models.ImageResult{
		Width:  width,
		Height: height,
		Seed:   seed,
		Prompt: req.Prompt,
		Model:  "nanobanana-2",
	}

	return result, imgData, nil
}

func (g *FalAIGenerator) submitToQueue(ctx context.Context, req models.GenerateRequest, seed int64, refURLs []string) (*falQueueResponse, error) {
	payload := map[string]any{
		"prompt":        req.Prompt,
		"seed":          seed,
		"output_format": falFormat(req.ImageFormat),
		"resolution":    falResolution(req.Width, req.Height),
		"num_images":    1,
	}

	// Use the /edit endpoint when reference images are provided.
	endpoint := g.cfg.FalAIModel
	if len(refURLs) > 0 {
		endpoint = g.cfg.FalAIModel + "/edit"
		payload["image_urls"] = refURLs
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s", falQueueBaseURL, endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+g.cfg.FalAIKey)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("queue submit returned %d: %s", resp.StatusCode, string(respBody))
	}

	var qr falQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return nil, fmt.Errorf("decoding queue response: %w", err)
	}
	return &qr, nil
}

func (g *FalAIGenerator) pollUntilDone(ctx context.Context, statusURL string) error {
	deadline := time.Now().Add(falMaxWait)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for fal.ai result")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(falPollInterval):
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		if err != nil {
			return err
		}
		httpReq.Header.Set("Authorization", "Key "+g.cfg.FalAIKey)

		resp, err := g.client.Do(httpReq)
		if err != nil {
			return err
		}

		var sr falStatusResponse
		json.NewDecoder(resp.Body).Decode(&sr)
		resp.Body.Close()

		g.logger.Debug("poll status", "status", sr.Status)

		if sr.Status == "COMPLETED" {
			return nil
		}
		if sr.Status != "IN_QUEUE" && sr.Status != "IN_PROGRESS" {
			return fmt.Errorf("unexpected status: %s", sr.Status)
		}
	}
}

func (g *FalAIGenerator) fetchResult(ctx context.Context, responseURL string) (*falResultResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, responseURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Key "+g.cfg.FalAIKey)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("result fetch returned %d: %s", resp.StatusCode, string(respBody))
	}

	// The result endpoint wraps the actual response in a "response" field.
	var wrapper struct {
		Response falResultResponse `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decoding result: %w", err)
	}

	return &wrapper.Response, nil
}

func (g *FalAIGenerator) downloadImage(ctx context.Context, url string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image download returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// uploadToFalStorage uploads image bytes to fal.ai's storage and returns a
// public URL that can be used with the /edit endpoint.
func (g *FalAIGenerator) uploadToFalStorage(ctx context.Context, data []byte, name string) (string, error) {
	filename := fmt.Sprintf("%s.png", name)

	// Step 1: Initiate upload to get a presigned URL.
	initPayload, _ := json.Marshal(map[string]any{
		"file_name":    filename,
		"content_type": "image/png",
	})
	initReq, err := http.NewRequestWithContext(ctx, http.MethodPost, falUploadURL, bytes.NewReader(initPayload))
	if err != nil {
		return "", err
	}
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("Authorization", "Key "+g.cfg.FalAIKey)

	initResp, err := g.client.Do(initReq)
	if err != nil {
		return "", err
	}
	defer initResp.Body.Close()

	if initResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(initResp.Body)
		return "", fmt.Errorf("fal storage initiate returned %d: %s", initResp.StatusCode, string(respBody))
	}

	var uploadInfo struct {
		FileURL   string `json:"file_url"`
		UploadURL string `json:"upload_url"`
	}
	if err := json.NewDecoder(initResp.Body).Decode(&uploadInfo); err != nil {
		return "", fmt.Errorf("decoding upload info: %w", err)
	}

	// Step 2: PUT the image data to the presigned URL.
	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadInfo.UploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	putReq.Header.Set("Content-Type", "image/png")

	putResp, err := g.client.Do(putReq)
	if err != nil {
		return "", err
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK && putResp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(putResp.Body)
		return "", fmt.Errorf("fal storage upload returned %d: %s", putResp.StatusCode, string(respBody))
	}

	g.logger.Debug("uploaded reference to fal storage", "name", name, "url", uploadInfo.FileURL)
	return uploadInfo.FileURL, nil
}

func falFormat(format string) string {
	switch format {
	case "jpg":
		return "jpeg"
	default:
		return "png"
	}
}

func falResolution(width, height int) string {
	maxDim := width
	if height > maxDim {
		maxDim = height
	}
	switch {
	case maxDim <= 512:
		return "0.5K"
	case maxDim <= 1024:
		return "1K"
	case maxDim <= 2048:
		return "2K"
	default:
		return "4K"
	}
}
