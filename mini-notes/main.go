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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()

	log.Println("Database connected")

	srv := Server{db: db}

	http.HandleFunc("GET /health", srv.healthHandler)
	http.HandleFunc("POST /notes", srv.createNoteHandler)
	http.HandleFunc("GET /notes", srv.listNoteHandler)

	log.Println("Server running on port :6767")
	log.Fatal(http.ListenAndServe(":6767", nil))
}
