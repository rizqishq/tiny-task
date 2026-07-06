package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	db *pgxpool.Pool
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

	log.Println("Database connected successfully")

	srv := &Server{db: db}

	http.HandleFunc("GET /health", srv.healthHandler)
	http.HandleFunc("POST /tasks", srv.createTaskHandler)
	http.HandleFunc("GET /tasks", srv.getTasksHandler)
	http.HandleFunc("GET /tasks/{id}", srv.getTaskByIDHandler)
	http.HandleFunc("PATCH /tasks/{id}", srv.updateTaskHandler)
	http.HandleFunc("DELETE /tasks/{id}", srv.deleteTaskHandler)

	log.Println("Server running on port :6969")
	log.Fatal(http.ListenAndServe(":6969", nil))
}
