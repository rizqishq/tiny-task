include .env

migrate-up:
	migrate -path migrations -database $(DATABASE_URL) up

migrate-down:
	migrate -path migrations -database $(DATABASE_URL) down 1

psql:
	docker exec -it go-postgres psql -U ${DATABASE_USER} -d ${DATABASE_NAME}

run:
	@go run .
