# TINY TASKS API

Simple task management api built with Go while also learning RESTful API in Go.

---

## How to run
1. Clone the repository
```bash
git clone https://github.com/rizqishq/tiny-task
cd tiny-task
```

2. Setup database
```bash
docker compose up -d
make migrate-up
```

3. Start the server
```go
make run
```

4. Test API endpoint

You can directly open [html documentation](docs/tiny-tasks-documentation.html) file in the browser and test it there.
