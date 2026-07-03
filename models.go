package main

import "time"

type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTaskRequest struct {
	Title string `json:"title"`
}
