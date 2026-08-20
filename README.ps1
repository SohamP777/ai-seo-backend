@'
# AI SEO Backend API

A powerful SEO optimization backend with AI capabilities.

## Features
- ✅ User authentication (JWT)
- ✅ SQLite database
- ✅ Rate limiting
- ✅ Workflow engine
- ✅ RESTful API

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register new user |
| POST | `/api/auth/login` | Login user |
| POST | `/api/workflows/start` | Start SEO workflow |
| GET | `/api/health` | Health check |
| GET | `/api/docs` | API documentation |

## Quick Start

1. Clone the repository
2. Copy `.env.example` to `.env`
3. Run `go mod tidy`
4. Run `go run main.go`

## Technologies
- Go 1.21+
- SQLite
- JWT Authentication
- Chi Router

## License
MIT
'@ | Out-File -FilePath README.md -Encoding UTF8

# Add and push README
git add README.md
git commit -m "Add README"
git push