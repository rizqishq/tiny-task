package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createNoteHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var req CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title can not be empty")
		return
	}

	query := `
	INSERT INTO notes (title, body)
	VALUES ($1, $2)
	RETURNING id, title, body, archived, created_at, updated_at
	`

	var note Note
	err := s.db.QueryRow(ctx, query, req.Title, req.Body).Scan(
		&note.ID,
		&note.Title,
		&note.Body,
		&note.Archived,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create note")
		return
	}

	writeJSON(w, http.StatusCreated, note)
}
