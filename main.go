package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("Failed to create connection pool: ", err)
	}
	defer db.Close()

	if err = db.Ping(ctx); err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	log.Println("Database connected succesfully")

	http.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		var req CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"message": "failed to read request body",
			})
			return
		}

		if req.Title == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"message": "title can not be empty",
			})
			return
		}

		query := `
		INSERT INTO tasks (title)
		VALUES ($1)
		RETURNING id, title, completed, created_at
		`

		var task Task

		err := db.QueryRow(ctx, query, req.Title).Scan(
			&task.ID,
			&task.Title,
			&task.Completed,
			&task.CreatedAt,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "failed to create task",
			})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"success": true,
			"message": "task saved successfully",
			"data":    task,
		})
	})

	http.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		query := `
		SELECT id, title, completed, created_at
		FROM tasks
		ORDER BY id DESC
		`

		rows, err := db.Query(ctx, query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "failed to retrieve data",
			})
			return
		}
		defer rows.Close()

		tasks := make([]Task, 0)
		for rows.Next() {
			var t Task
			err := rows.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"success": false,
					"message": "failed to scan data",
				})
				return
			}

			tasks = append(tasks, t)
		}

		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "failed to retrieve tasks",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "tasks retrieved successfully",
			"data":    tasks,
		})
	})

	http.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"message": "invalid task id",
			})
			return
		}

		query := `
		SELECT id, title, completed, created_at
		FROM tasks
		WHERE id=$1
		`

		var task Task

		err = db.QueryRow(ctx, query, id).Scan(
			&task.ID,
			&task.Title,
			&task.Completed,
			&task.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]any{
					"success": false,
					"message": "task not found",
				})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"success": false,
					"message": "operation failed",
				})
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "task retrieved successfully",
			"data":    task,
		})
	})

	http.HandleFunc("PATCH /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"message": "invalid task id",
			})
			return
		}

		query := `
		UPDATE tasks
		SET title = $1, completed = $2
		WHERE id = $3
		RETURNING id, title, completed, created_at
		`

		var req UpdateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"message": "invalid request",
			})
			return
		}

		if req.Title == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"message": "title can not be empty",
			})
			return
		}

		var task Task
		err = db.QueryRow(ctx, query, req.Title, req.Completed, id).Scan(
			&task.ID,
			&task.Title,
			&task.Completed,
			&task.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]any{
					"success": false,
					"message": "update failed",
				})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"success": false,
					"message": "update failed",
				})
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "task updated",
			"data":    task,
		})
	})

	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"success": false,
				"message": "database is asleep",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "server up",
		})
	})

	log.Println("Server running on port :6969")
	log.Fatal(http.ListenAndServe(":6969", nil))
}
