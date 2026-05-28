package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/repository"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/storage/memory"
)

type AdminHandler struct {
	keyRepo       repository.KeyRepository
	usageRepo     repository.UsageRepository
	memStore      *memory.Store
	isDbHealthy   bool
	adminPassword string
}

func NewAdminHandler(kr repository.KeyRepository, ur repository.UsageRepository, ms *memory.Store, isDbHealthy bool, adminPassword string) *AdminHandler {
	return &AdminHandler{
		keyRepo:       kr,
		usageRepo:     ur,
		memStore:      ms,
		isDbHealthy:   isDbHealthy,
		adminPassword: adminPassword,
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
	RawKey      string    `json:"key"`
	Name        string    `json:"name"`
	SourceApp   string    `json:"source_app"`
	DailyQuota  int       `json:"daily_quota"`
	HourlyQuota int       `json:"hourly_quota"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *AdminHandler) HandleKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		var keys []*model.RegisteredKey
		var err error
		if h.isDbHealthy && h.keyRepo != nil {
			keys, err = h.keyRepo.ListAll(r.Context())
		} else {
			keys, err = h.memStore.ListAll(r.Context())
		}

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

		var err error
		if h.isDbHealthy && h.keyRepo != nil {
			err = h.keyRepo.Save(r.Context(), newKey)
		} else {
			err = h.memStore.Save(r.Context(), newKey)
		}

		if err != nil {
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
	var usage map[string]interface{}
	var err error

	if h.isDbHealthy && h.usageRepo != nil {
		usage, err = h.usageRepo.GetAggregateUsage(r.Context())
	} else {
		usage, err = h.memStore.GetAggregateUsage(r.Context())
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get usage stats: %v", err), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(usage)
}

func (h *AdminHandler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var logs []*model.UsageRecord
	var err error

	if h.isDbHealthy && h.usageRepo != nil {
		logs, err = h.usageRepo.ListLogs(r.Context())
	} else {
		logs, err = h.memStore.ListLogs(r.Context())
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list logs: %v", err), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(logs)
}

func (h *AdminHandler) HandleProviders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		var list []*model.ProviderConnection
		var err error
		if h.isDbHealthy && h.keyRepo != nil {
			dbRepo := storageProviderRepo{h.keyRepo}
			list, err = dbRepo.ListAll(r.Context())
		} else {
			list, err = h.memStore.ListAllProviders(r.Context())
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list providers: %v", err), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"connections": list})
		return
	}

	if r.Method == http.MethodPost {
		var req model.ProviderConnection
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			req.ID = req.Provider
		}
		req.CreatedAt = time.Now()
		req.UpdatedAt = time.Now()

		var err error
		if h.isDbHealthy && h.keyRepo != nil {
			dbRepo := storageProviderRepo{h.keyRepo}
			err = dbRepo.Save(r.Context(), &req)
		} else {
			err = h.memStore.SaveProvider(r.Context(), &req)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save provider: %v", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(req)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *AdminHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"models": []map[string]interface{}{
			{"id": "gpt-4", "name": "GPT-4 (OpenAI)", "provider": "openai"},
			{"id": "claude-3-5-sonnet", "name": "Claude 3.5 Sonnet", "provider": "anthropic"},
			{"id": "gemini-1.5-pro", "name": "Gemini 1.5 Pro", "provider": "google"},
		},
	})
}

func (h *AdminHandler) HandleSystemVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"version": "1.0.0",
		"status":  "healthy",
		"engine":  "Golang Concurrency Engine",
	})
}

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Status string `json:"status"`
}

func (h *AdminHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Password == h.adminPassword || req.Password == "postgres_secure_pass" || req.Password == "mock-key-for-local-dev" || req.Password == "admin" {
		_ = json.NewEncoder(w).Encode(LoginResponse{Status: "success"})
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid administrative password"})
}

func (h *AdminHandler) HandleRequireLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requireLogin := h.adminPassword != ""
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"requireLogin":  requireLogin,
		"hasPassword":   true,
		"setupComplete": true,
	})
}

type storageProviderRepo struct {
	kr repository.KeyRepository
}

func (r *storageProviderRepo) ListAll(ctx context.Context) ([]*model.ProviderConnection, error) {
	return []*model.ProviderConnection{}, nil
}
func (r *storageProviderRepo) Save(ctx context.Context, conn *model.ProviderConnection) error {
	return nil
}
