package travelagent

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Handler struct {
	agent *TravelAdvisorAgent
}

func NewHandler(agent *TravelAdvisorAgent) *Handler {
	return &Handler{agent: agent}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/travel-advisor/stream", h.handleStream)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request) {
	writeCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req AdviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json request"})
		return
	}
	if req.Message == "" && req.Destination == "" && req.TravelDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message, destination or travelDate is required"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	emit := func(event StreamEvent) error {
		data, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return err
		}

		flusher.Flush()
		return nil
	}

	if err := h.agent.StreamAdvice(r.Context(), req, emit); err != nil {
		_ = emit(StreamEvent{
			Type: "error",
			Data: map[string]string{"message": err.Error()},
		})
	}
}

func writeCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
