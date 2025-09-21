package httpapi

import (
	"encoding/json"
	"errors"
	"gopool/internal/queue"
	"io"
	"net/http"
	"strings"
)

type API struct {
	Q     *queue.Q
	Store queue.Store
}

func New(q *queue.Q, s queue.Store) *API {
	return &API{Q: q, Store: s}
}

func (a *API) NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/enqueue", a.enqueue)
	mux.HandleFunc("/healthz", a.healthz)
	return mux
}

func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a *API) enqueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB safety cap
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req EnqueueRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := validateEnqueue(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	t := queue.Task{
		ID:         req.ID,
		Payload:    req.Payload,
		MaxRetries: req.MaxRetries,
	}
	if err := a.Q.Enqueue(t); err != nil {
		switch err {
		case queue.ErrDuplicateId:
			writeError(w, http.StatusConflict, "duplicate id")
			return
		case queue.ErrQueueFull, queue.ErrClosed:
			// Not accepting / full: tell callers to retry later
			writeError(w, http.StatusServiceUnavailable, "queue unavailable")
			return
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	resp := EnqueueResponse{ID: req.ID, State: string(queue.StateQueued)}
	writeJSON(w, http.StatusAccepted, resp) // 202 Accepted
}

func validateEnqueue(r EnqueueRequest) error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("id is required")
	}

	if r.MaxRetries < 0 {
		return errors.New("max_retries must be >= 0")
	}
	return nil
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, POST")
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
