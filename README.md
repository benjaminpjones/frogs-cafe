# Frogs Café - Go Game Server

A WebSocket-enabled Go (Baduk/Weiqi) game server built with Go and PostgreSQL, featuring a React frontend.

## Project Structure

```
frogs_cafe/
├── server/                # Go backend
├── web_client/            # React frontend application
├── docker-compose.yml     # Production Docker composition
├── docker-compose.dev.yml # Development Docker composition
├── Dockerfile             # Production container image
├── Dockerfile.dev         # Development container image
├── README.md
└── .env                   # Environment variables (not in git)
```

## Features

- **WebSocket Support**: Real-time game updates and communication
- **REST API**: Full CRUD operations for games and players
- **PostgreSQL Database**: Persistent storage with automatic migrations
- **React Frontend**: Interactive game board with real-time updates
- **Docker Support**: Easy development environment setup

## Prerequisites

- Go 1.24 or later
- PostgreSQL 15 or later (or use Docker)
- Node.js 16 or later (for frontend)
- Docker (with Docker Compose V2 plugin) - optional, for containerized development

## Getting Started

### Development Mode (Recommended)

**Best for active development with hot-reload:**

1. **Start backend services with Docker:**
   ```bash
   docker compose -f docker-compose.dev.yml up
   ```
   This starts PostgreSQL and the Go server on `localhost:8080`

2. **Run React dev server locally:**
   ```bash
   cd web_client
   npm install
   npm run dev
   ```
   Frontend runs on `localhost:3000` with hot-reload. API calls are automatically proxied to the backend.

### Production Mode

**Single container with built React app:**

```bash
docker compose up --build
```

This builds the React app into static files and serves them from the Go server at `localhost:8080`

### Manual Setup (No Docker)

1. **Set up PostgreSQL**:
   ```bash
   createdb frogs_cafe
   ```

2. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your database connection string
   ```

3. **Run the server**:
   ```bash
   cd server
   go mod download
   go run main.go
   ```

4. **Run the client**:
   ```bash
   cd web_client
   npm install
   npm run dev
   ```

## API Endpoints

### REST API

- `GET /health` - Health check
- `POST /api/v1/register` - Register a new user
- `POST /api/v1/login` - Login with username and password
- `POST /api/v1/logout` - Logout (invalidates session)
- `GET /api/v1/games` - List all games
- `POST /api/v1/games` - Create a new game
- `GET /api/v1/games/{gameID}` - Get game details
- `GET /api/v1/games/{gameID}/moves` - Get all moves for a game
- `GET /api/v1/players` - List all players
- `POST /api/v1/players` - Create a new player
- `GET /api/v1/players/{playerID}` - Get player details

### WebSocket

- `WS /ws?game_id={gameID}&user_id={userID}` - Connect to game updates

## Database Schema

### Players Table
- `id`: Serial primary key
- `username`: Unique username
- `email`: Unique email
- `password_hash`: Hashed password (bcrypt)
- `rating`: Player rating (default 1500)
- `created_at`, `updated_at`: Timestamps

### Games Table
- `id`: Serial primary key
- `black_player_id`, `white_player_id`: Player references
- `board_size`: Board dimensions (default 19)
- `status`: Game status (waiting/active/finished)
- `winner_id`: Winner reference
- `created_at`, `updated_at`: Timestamps

### Moves Table
- `id`: Serial primary key
- `game_id`: Game reference
- `player_id`: Player reference
- `move_number`: Sequential move number
- `x`, `y`: Board coordinates
- `created_at`: Timestamp

### Sessions Table
- `id`: Serial primary key
- `player_id`: Player reference
- `token`: Unique session token
- `last_activity`: Last activity timestamp (for sliding window)
- `expires_at`: Session expiration time
- `created_at`: Timestamp

## Development

### Development Tools

**Go Linting (Optional for Local Development)**

The CI pipeline uses `golangci-lint` to ensure code quality. While not required for local development, you can install it to catch issues before pushing:

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

**Run the linter:**
```bash
cd server
golangci-lint run --timeout=5m
```

> **Note:** Do not use `go install` to install golangci-lint as it can cause dependency conflicts. See the [official installation guide](https://golangci-lint.run/welcome/install/) for more options.

**Frontend Formatting**

The project uses Prettier for code formatting:

```bash
cd web_client
npm run format        # Format all files
npm run format:check  # Check formatting without modifying
```

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
# Build backend
cd server
go build -o frogs-cafe-server main.go

# Build frontend
cd web_client
npm run build
```

### Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `PORT`: Server port (default: 8080)
- `ENVIRONMENT`: Environment mode (development/production)

## Technology Stack

### Backend
- **Go**: Core server language
- **Chi**: HTTP router
- **Gorilla WebSocket**: WebSocket implementation
- **PostgreSQL**: Database
- **godotenv**: Environment configuration

### Frontend
- **React**: UI framework
- **TypeScript**: Type-safe JavaScript
- **Canvas API**: Game board rendering
- **WebSocket API**: Real-time communication

## Contributing

1. Create a feature branch
2. Make your changes
3. Run tests
4. Submit a pull request

## License

MIT License

## Future Enhancements

- [x] User authentication and authorization
- [ ] Game rules engine (capture, ko, scoring)
- [ ] Game replay functionality
- [ ] Chat system
- [ ] Rating system (ELO)
- [ ] Tournament support
- [ ] AI opponent integration
