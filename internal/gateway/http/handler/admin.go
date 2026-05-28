package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/repository"
)

type AdminHandler struct {
	keyRepo   repository.KeyRepository
	usageRepo repository.UsageRepository
}

func NewAdminHandler(kr repository.KeyRepository, ur repository.UsageRepository) *AdminHandler {
	return &AdminHandler{
		keyRepo:   kr,
		usageRepo: ur,
	}
}

type CreateKeyRequest struct {
	Name        string `json:"name"`
	SourceApp   string `json:"source_app"`
	DailyQuota  int    `json:"daily_quota"`
	HourlyQuota int    `json:"hourly_quota"`
}

type CreateKeyResponse struct {
	ID          string    `json:"id"`
	RawKey      string    `json:"key"` // Raw key is only visible once
	Name        string    `json:"name"`
	SourceApp   string    `json:"source_app"`
	DailyQuota  int       `json:"daily_quota"`
	HourlyQuota int       `json:"hourly_quota"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *AdminHandler) HandleKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		keys, err := h.keyRepo.ListAll(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list keys: %v", err), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(keys)
		return
	}

	if r.Method == http.MethodPost {
		var req CreateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Generate cryptographically secure unique key
		rawBytes := make([]byte, 16)
		_, _ = rand.Read(rawBytes)
		rawKey := "ork_" + hex.EncodeToString(rawBytes)
		keyHash := auth.HashKey(rawKey)
		keyID := "key_" + hex.EncodeToString(rawBytes[:8])

		newKey := &model.RegisteredKey{
			ID:          keyID,
			KeyHash:     keyHash,
			Name:        req.Name,
			SourceApp:   req.SourceApp,
			DailyQuota:  req.DailyQuota,
			HourlyQuota: req.HourlyQuota,
			Active:      true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := h.keyRepo.Save(r.Context(), newKey); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save key: %v", err), http.StatusInternalServerError)
			return
		}

		resp := CreateKeyResponse{
			ID:          newKey.ID,
			RawKey:      rawKey,
			Name:        newKey.Name,
			SourceApp:   newKey.SourceApp,
			DailyQuota:  newKey.DailyQuota,
			HourlyQuota: newKey.HourlyQuota,
			CreatedAt:   newKey.CreatedAt,
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *AdminHandler) HandleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	usage, err := h.usageRepo.GetAggregateUsage(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get usage stats: %v", err), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(usage)
}
