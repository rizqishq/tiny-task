package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{
			Success: false,
			Message: "database is asleep",
		})
		return
	}
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "server up",
	})
}

func (s *Server) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Message: "failed to read request body",
		})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Message: "title can not be empty",
		})
		return
	}

	query := `
		INSERT INTO tasks (title)
		VALUES ($1)
		RETURNING id, title, completed, created_at
		`

	var task Task

	err := s.db.QueryRow(ctx, query, req.Title).Scan(
		&task.ID,
		&task.Title,
		&task.Completed,
		&task.CreatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Message: "failed to create task",
		})
		return
	}

	writeJSON(w, http.StatusCreated, Response{
		Success: true,
		Message: "task saved successfully",
		Data:    task,
	})
}

func (s *Server) getTasksHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	query := `
		SELECT id, title, completed, created_at
		FROM tasks
		ORDER BY id DESC
		`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Message: "failed to retrieve data",
		})
		return
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		err := rows.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{
				Success: false,
				Message: "failed to scan data",
			})
			return
		}

		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Message: "failed to retrieve tasks",
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "tasks retrieved successfully",
		Data:    tasks,
	})
}

func (s *Server) getTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	query := `
		SELECT id, title, completed, created_at
		FROM tasks
		WHERE id = $1
		`

	var task Task

	err := s.db.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		&task.Completed,
		&task.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, Response{
				Success: false,
				Message: "task not found",
			})
		} else {
			writeJSON(w, http.StatusInternalServerError, Response{
				Success: false,
				Message: "operation failed",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "task retrieved successfully",
		Data:    task,
	})
}

func (s *Server) updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Message: "invalid request",
		})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Message: "title can not be empty",
		})
		return
	}

	query := `
		UPDATE tasks
		SET title = $1, completed = $2
		WHERE id = $3
		RETURNING id, title, completed, created_at
		`

	var task Task

	err := s.db.QueryRow(ctx, query, req.Title, req.Completed, id).Scan(
		&task.ID,
		&task.Title,
		&task.Completed,
		&task.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, Response{
				Success: false,
				Message: "task not found",
			})
		} else {
			writeJSON(w, http.StatusInternalServerError, Response{
				Success: false,
				Message: "update failed",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "task updated",
		Data:    task,
	})
}

func (s *Server) deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}

	result, err := s.db.Exec(ctx, "DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Message: "failed to delete task",
		})
		return
	}

	if result.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, Response{
			Success: false,
			Message: "task not found",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
