package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vinq1911/nanobanana-image-tool/internal/config"
	"github.com/vinq1911/nanobanana-image-tool/internal/generator"
	"github.com/vinq1911/nanobanana-image-tool/internal/models"
	"github.com/vinq1911/nanobanana-image-tool/internal/storage"
)

// Server is the HTTP API server for nanobanana-image-tool.
type Server struct {
	cfg       *config.Config
	generator generator.ImageGenerator
	storage   storage.Storage
	logger    *slog.Logger
	srv       *http.Server
}

// NewServer creates a new HTTP API server.
func NewServer(cfg *config.Config, gen generator.ImageGenerator, store storage.Storage, logger *slog.Logger) *Server {
	s := &Server{
		cfg:       cfg,
		generator: gen,
		storage:   store,
		logger:    logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /generate", s.handleGenerate)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /tool-schema", s.handleToolSchema)

	s.srv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 180 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	s.logger.Info("starting HTTP server", "addr", s.srv.Addr)
	return s.srv.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req models.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = s.cfg.OutputDir
	}

	result, imgData, err := s.generator.Generate(r.Context(), req)
	if err != nil {
		s.logger.Error("generation failed", "error", err)
		s.writeError(w, http.StatusInternalServerError, "generation failed", err.Error())
		return
	}

	imgPath, err := s.storage.Save(r.Context(), imgData, req.ImageFormat, outputDir)
	if err != nil {
		s.logger.Error("storage failed", "error", err)
		s.writeError(w, http.StatusInternalServerError, "failed to save image", err.Error())
		return
	}

	result.ImagePath = imgPath

	resp := models.GenerateResponse{
		ImagePath: imgPath,
		Metadata:  *result,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) writeError(w http.ResponseWriter, status int, msg, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: msg, Details: details})
}
