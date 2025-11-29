# Frogs Café - Go Server Project

This is a Go server project with WebSocket support, REST API, PostgreSQL database, and React frontend for a Go (Baduk/Weiqi) game server.

## Architecture
- Backend: Go with Gorilla WebSocket and Chi router (in `server/`)
- Database: PostgreSQL
- Frontend: React with TypeScript and Vite (in `web_client/`)
- Containerization: Docker Compose

## Project Structure
- `server/` - All Go backend code (main.go, config/, database/, handlers/, middleware/, models/)
- `web_client/` - All React frontend code
- `scripts/` - Database initialization scripts
- Docker files at root level

## Development Guidelines
- Follow Go best practices and idiomatic patterns
- Use proper error handling and logging
- Implement clean architecture with separation of concerns
- Write tests for critical functionality
- Use environment variables for configuration
