package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Server) createBookHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var req CreateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AuthorID <= 0 {
		writeError(w, http.StatusBadRequest, "author_id cant be empty")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title cant be empty")
		return
	}

	query := `
		WITH new_book AS (
			INSERT INTO books (author_id, title, description)
			VALUES ($1, $2, $3)
			RETURNING id, author_id, title, description, status, created_at, updated_at
		)
		SELECT
			nb.id, nb.title, nb.description, nb.status, nb.created_at, nb.updated_at,
			a.id, a.name
		FROM new_book nb
		JOIN authors a ON a.id = nb.author_id
	`

	var b Book
	err := s.db.QueryRow(ctx, query, req.AuthorID, req.Title, req.Description).Scan(
		&b.ID,
		&b.Title,
		&b.Description,
		&b.Status,
		&b.CreatedAt,
		&b.UpdatedAt,
		&b.Author.ID,
		&b.Author.Name,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			writeError(w, http.StatusBadRequest, "author does not exist")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create book")
		return
	}

	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) getAllBooksHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	query := `
	SELECT
			b.id, b.title, b.description, b.status, b.created_at, b.updated_at,
			a.id, a.name
	FROM books b
	JOIN authors a ON a.id = b.author_id
	ORDER BY b.created_at DESC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch books")
		return
	}
	defer rows.Close()

	books := make([]Book, 0)
	for rows.Next() {
		var b Book
		if err := rows.Scan(
			&b.ID,
			&b.Title,
			&b.Description,
			&b.Status,
			&b.CreatedAt,
			&b.UpdatedAt,
			&b.Author.ID,
			&b.Author.Name,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan book")
			return
		}
		books = append(books, b)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "error while reading books")
		return
	}

	writeJSON(w, http.StatusOK, books)
}

func (s *Server) getBookByIdHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	query := `
	SELECT
			b.id, b.title, b.description, b.status, b.created_at, b.updated_at,
			a.id, a.name
	FROM books b
	JOIN authors a ON a.id = b.author_id
	WHERE b.id = $1
	`

	var b Book
	err = s.db.QueryRow(ctx, query, id).Scan(
		&b.ID,
		&b.Title,
		&b.Description,
		&b.Status,
		&b.CreatedAt,
		&b.UpdatedAt,
		&b.Author.ID,
		&b.Author.Name,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "book not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch data")
		return
	}

	writeJSON(w, http.StatusOK, b)
}

func (s *Server) updateBookHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req UpdateBookRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AuthorID == nil && req.Title == nil && req.Description == nil {
		writeError(w, http.StatusBadRequest, "at least one field is required")
		return
	}
	if req.Title != nil && *req.Title == "" {
		writeError(w, http.StatusBadRequest, "title cannot be empty")
		return
	}

	query := `
	WITH updated_book AS (
		UPDATE books
		SET
			author_id = COALESCE($1, author_id),
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			updated_at = NOW()
		WHERE id = $4
		RETURNING id, title, description, status, created_at, updated_at, author_id
	)
	SELECT
		ub.id, ub.title, ub.description, ub.status, ub.created_at, ub.updated_at,
		a.id, a.name
	FROM updated_book ub
	JOIN authors a ON a.id = ub.author_id
	`

	var b Book
	err = s.db.QueryRow(ctx, query, req.AuthorID, req.Title, req.Description, id).Scan(
		&b.ID,
		&b.Title,
		&b.Description,
		&b.Status,
		&b.CreatedAt,
		&b.UpdatedAt,
		&b.Author.ID,
		&b.Author.Name,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			writeError(w, http.StatusBadRequest, "author does not exist")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "book not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update book")
		return
	}

	writeJSON(w, http.StatusOK, b)
}

func (s *Server) updateBookStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req UpdateBookStatusRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !validStatus[req.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	query := `
	WITH updated_book AS (
		UPDATE books
		SET
			status = $1,
			updated_at = NOW()
		WHERE id = $2
		RETURNING id, title, description, status, created_at, updated_at, author_id
	)
	SELECT
		ub.id, ub.title, ub.description, ub.status, ub.created_at, ub.updated_at,
		a.id, a.name
	FROM updated_book ub
	JOIN authors a ON a.id = ub.author_id
	`

	var b Book
	err = s.db.QueryRow(ctx, query, req.Status, id).Scan(
		&b.ID,
		&b.Title,
		&b.Description,
		&b.Status,
		&b.CreatedAt,
		&b.UpdatedAt,
		&b.Author.ID,
		&b.Author.Name,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			writeError(w, http.StatusBadRequest, "author does not exist")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "book not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update book")
		return
	}

	writeJSON(w, http.StatusOK, b)
}

func (s *Server) deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	query := `
	DELETE FROM books
	WHERE id = $1
	`

	tag, err := s.db.Exec(ctx, query, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete book")
		return
	}

	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "books not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
