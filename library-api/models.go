package main

import "time"

type Author struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type BookAuthor struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Book struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Author      BookAuthor `json:"author"`
}

type CreateAuthorRequest struct {
	Name string `json:"name"`
}

type CreateBookRequest struct {
	AuthorID    int64  `json:"author_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateBookRequest struct {
	AuthorID    *int64  `json:"author_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type UpdateBookStatusRequest struct {
	Status string `json:"status"`
}
