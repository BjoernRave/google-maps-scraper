package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gosom/google-maps-scraper/gmaps"
	"go.uber.org/zap"
)

// JobHandler handles HTTP requests for job operations
type JobHandler struct {
	provider gmaps.Provider
	logger   *zap.Logger
}

// NewJobHandler creates a new JobHandler instance
func NewJobHandler(provider gmaps.Provider, logger *zap.Logger) *JobHandler {
	return &JobHandler{
		provider: provider,
		logger:   logger,
	}
}

type CreateJobRequest struct {
	Query        string `json:"query"`
	Language     string `json:"language"`
	MaxDepth     int    `json:"max_depth"`
	ExtractEmail bool   `json:"extract_email"`
	GeoCoords    string `json:"geo_coordinates"`
	Zoom         int    `json:"zoom"`
}

type CreateJobResponse struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id"`
}

func (r *CreateJobRequest) validate() error {
	var errors []string

	if strings.TrimSpace(r.Query) == "" {
		errors = append(errors, "query is required")
	}

	if strings.TrimSpace(r.Language) == "" {
		errors = append(errors, "language is required")
	}

	if r.MaxDepth < 0 || r.MaxDepth > 10 {
		errors = append(errors, "max_depth must be between 0 and 10")
	}

	if r.Zoom < 0 || r.Zoom > 21 {
		errors = append(errors, "zoom must be between 0 and 21")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	return nil
}

// CreateJob handles the creation of new scraping jobs
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	logger := h.logger.With(
		zap.String("request_id", requestID),
		zap.String("handler", "CreateJob"),
	)

	// Verify HTTP method
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", requestID)
		return
	}

	// Parse request body
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("failed to decode request body", zap.Error(err))
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body", requestID)
		return
	}

	// Validate request
	if err := req.validate(); err != nil {
		logger.Error("request validation failed", zap.Error(err))
		h.respondWithError(w, http.StatusBadRequest, err.Error(), requestID)
		return
	}

	// Create job
	jobID := uuid.New().String()
	job := gmaps.NewGmapJob(
		jobID,
		req.Language,
		req.Query,
		req.MaxDepth,
		req.ExtractEmail,
		req.GeoCoords,
		req.Zoom,
	)

	// Push job to provider
	if err := h.provider.Push(r.Context(), job); err != nil {
		logger.Error("failed to push job",
			zap.Error(err),
			zap.String("job_id", jobID),
		)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create job", requestID)
		return
	}

	logger.Info("job created successfully",
		zap.String("job_id", jobID),
		zap.String("query", req.Query),
	)

	// Respond with success
	h.respondWithJSON(w, http.StatusCreated, CreateJobResponse{
		JobID:     jobID,
		Status:    "created",
		Message:   "Job created successfully",
		RequestID: requestID,
	})
}

func (h *JobHandler) respondWithError(w http.ResponseWriter, code int, message string, requestID string) {
	h.respondWithJSON(w, code, CreateJobResponse{
		Status:    "error",
		Message:   message,
		RequestID: requestID,
	})
}

func (h *JobHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
