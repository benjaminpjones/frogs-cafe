# Authentication

Session-based authentication with sliding window expiration for the Frogs Café Go server.

## Endpoints

### Register
```bash
POST /api/v1/register
Content-Type: application/json

{
  "username": "player1",
  "email": "player1@example.com",
  "password": "password123"
}
```

### Login
```bash
POST /api/v1/login
Content-Type: application/json

{
  "username": "player1",
  "password": "password123"
}
```

Both endpoints return:
```json
{
  "token": "base64-encoded-session-token",
  "player": { ... }
}
```

### Logout
```bash
POST /api/v1/logout
Authorization: Bearer YOUR_SESSION_TOKEN
```

## Using the Session Token

### REST API
Include the token in the Authorization header:
```bash
Authorization: Bearer YOUR_SESSION_TOKEN
```

### WebSocket
Pass the token as a query parameter:
```
ws://localhost:8080/ws?game_id=1&token=YOUR_SESSION_TOKEN
```

## Session Behavior

- **Duration**: Sessions last 7 days from last activity
- **Sliding Window**: Session expiration extends by 7 days on any activity (if >30 minutes since last activity)
- **Cleanup**: Expired sessions are automatically removed every hour
- **Logout**: Immediately invalidates the session
