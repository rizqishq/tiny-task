package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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

func (s *Server) getAllNotesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	notes := make([]Note, 0)
	query := `
	SELECT id, title, body, archived, created_at, updated_at
	FROM notes
	ORDER BY id DESC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get notes")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var n Note
		err := rows.Scan(&n.ID, &n.Title, &n.Body, &n.Archived, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan note")
			return
		}
		notes = append(notes, n)
	}

	writeJSON(w, http.StatusOK, notes)
}

func (s *Server) getNoteByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	query := `
	SELECT id, title, body, archived, created_at, updated_at
	FROM notes
	WHERE id = $1
	`

	var note Note
	err = s.db.QueryRow(ctx, query, id).Scan(
		&note.ID,
		&note.Title,
		&note.Body,
		&note.Archived,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "operation failed")
		return
	}

	writeJSON(w, http.StatusOK, note)
}
