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

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createAuthorHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var req CreateAuthorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name cant be empty")
		return
	}

	query := `
	INSERT INTO authors (name)
	VALUES ($1)
	RETURNING id, name, created_at
	`

	var author Author
	err := s.db.QueryRow(ctx, query, req.Name).Scan(
		&author.ID,
		&author.Name,
		&author.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "author already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create author")
		return
	}

	writeJSON(w, http.StatusCreated, author)
}

func (s *Server) getAllAuthorsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	query := `
	SELECT id, name, created_at
	FROM authors
	ORDER BY name
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch authors")
		return
	}
	defer rows.Close()

	authors := make([]Author, 0)
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan author")
			return
		}
		authors = append(authors, a)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "error while reading author")
		return
	}

	writeJSON(w, http.StatusOK, authors)
}

func (s *Server) getAuthorByIdHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	query := `
	SELECT id, name, created_at
	FROM authors
	WHERE id = $1
	`

	var a Author
	err = s.db.QueryRow(ctx, query, id).Scan(
		&a.ID,
		&a.Name,
		&a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "author not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch author")
		return
	}

	writeJSON(w, http.StatusOK, a)
}

func (s *Server) deleteAuthorHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	query := `
	DELETE FROM authors
	WHERE id = $1
	`

	tag, err := s.db.Exec(ctx, query, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23001" {
			writeError(w, http.StatusConflict, "cannot delete author: still has related books")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete author")
		return
	}

	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "author not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

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
	SELECT b.id, b.title, b.description, b.status, b.created_at, b.updated_at, a.id, a.name
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
	SELECT b.id, b.title, b.description, b.status, b.created_at, b.updated_at, a.id, a.name
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
