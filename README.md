# Ticket System Backend Service

A complete, production-grade Go-based backend service for a "Ticket System". This service provides user registration, authentication, ticket creation, list filtering (users can only access their own tickets), and strict status transitions.

## Features & Implementation Highlights

- **Language & Framework**: Built in **Go** utilizing standard `net/http` and `github.com/go-chi/chi/v5` for lightweight, idiomatic routing.
- **Persistence**: Powered by **SQLite** using the pure Go driver `modernc.org/sqlite` with standard `database/sql`. Being a CGO-free implementation, it builds cleanly in a lightweight `alpine` container without needing host compiler/GCC dependencies.
- **Authentication**: JWT-based auth via `github.com/golang-jwt/jwt/v5`, passed via the standard `Authorization: Bearer <token>` header.
- **Password Security**: Passwords are securely hashed with `bcrypt` (`golang.org/x/crypto/bcrypt`) before saving to SQLite.
- **ownership Checks**: Custom middleware and handlers validate that users can only retrieve or update status for tickets they created.
- **Status Transitions Enforced**:
  - Valid statuses: `open`, `in_progress`, `closed`.
  - Allowed sequence: `open` -> `in_progress` -> `closed`.
  - Reopening closed tickets or skipping states (e.g. `open` -> `closed` directly) is blocked and returns a `400 Bad Request`.
- **Dockerized**: Containerized with a multi-stage Docker build resulting in a minimal, highly secure final image.

---

## Project Structure

```
/cmd/server/main.go        # Server entrypoint and router registration
/internal/auth/            # password hashing & JWT generation/validation
/internal/handlers/        # HTTP controllers/handlers & transition rules
/internal/middleware/      # Authentication middleware injecting user claims to request context
/internal/models/          # Structs for db mapping and JSON requests/responses
/internal/store/           # SQLite store wrapper & table migrations
/schema.sql                # SQL schema reference
Dockerfile                 # Multi-stage production container build
.env.example               # Config template
go.mod / go.sum            # Dependencies configuration
```

---

## Configuration & Environment Variables

The application is configured using environment variables. You can specify them in the shell or place them inside a `.env` file in the project root directory.

Refer to [.env.example](.env.example) for placeholder values:
- `PORT`: The port the service runs on (defaults to `8080`).
- `JWT_SECRET`: Secret string used for signing and verifying JSON Web Tokens.
- `DB_PATH`: SQLite database file path (defaults to `tickets.db`).

---

## Local Run Instructions

### 1. Run using standard Go (locally)
Ensure you have Go installed on your machine.
```bash
# Clone/Open the directory, then download dependencies
go mod download

# Run the server
go run cmd/server/main.go
```
The server will start on port `8080` (or the port defined in your `.env` file).

### 2. Run using Docker (Containerized)
Build the image and spin up the container:
```bash
# Build the Docker image
docker build -t ticket-system .

# Run the container (mounting volume for persistent db storage)
docker run -p 8080:8080 -v ticket_data:/data -e DB_PATH=/data/tickets.db ticket-system
```

Verify that the server is running by hitting the health check endpoint:
```bash
curl http://localhost:8080/health
```

---

## API Documentation

### 1. Health Check
* **Method**: `GET`
* **Path**: `/health`
* **Auth Required**: No (Public)
* **Response Code**: `200 OK`
* **Response Body**:
  ```json
  {
    "status": "ok"
  }
  ```

---

### 2. User Registration
* **Method**: `POST`
* **Path**: `/auth/register`
* **Auth Required**: No (Public)
* **Request Body**:
  ```json
  {
    "username": "alice",
    "email": "alice@example.com",
    "password": "securepassword123"
  }
  ```
* **Success Response (201 Created)**:
  ```json
  {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com"
  }
  ```
* **Error Response (400 Bad Request)**: Missing fields.
* **Error Response (409 Conflict)**: Duplicate username or email.

---

### 3. User Login
* **Method**: `POST`
* **Path**: `/auth/login`
* **Auth Required**: No (Public)
* **Request Body**:
  ```json
  {
    "username": "alice",
    "password": "securepassword123"
  }
  ```
  *(Note: You can also login by passing the `"email"` field instead of `"username"`)*
* **Success Response (200 OK)**:
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsIn...",
    "jwt": "eyJhbGciOiJIUzI1NiIsIn..."
  }
  ```
* **Error Response (401 Unauthorized)**: Invalid credentials.

---

### 4. Create Ticket
* **Method**: `POST`
* **Path**: `/tickets`
* **Auth Required**: Yes (Bearer Token)
* **Headers**: `Authorization: Bearer <JWT_TOKEN>`
* **Request Body**:
  ```json
  {
    "title": "Fix Database Timeout",
    "description": "Database locks up on heavy concurrent requests"
  }
  ```
* **Success Response (201 Created)**:
  ```json
  {
    "id": 1,
    "title": "Fix Database Timeout",
    "description": "Database locks up on heavy concurrent requests",
    "status": "open",
    "owner_id": 1,
    "user_id": 1,
    "created_at": "2026-07-16T14:45:00Z",
    "updated_at": "2026-07-16T14:45:00Z"
  }
  ```
* **Error Response (401 Unauthorized)**: Missing or invalid token.

---

### 5. List Tickets
* **Method**: `GET`
* **Path**: `/tickets`
* **Auth Required**: Yes (Bearer Token)
* **Headers**: `Authorization: Bearer <JWT_TOKEN>`
* **Success Response (200 OK)**:
  ```json
  [
    {
      "id": 1,
      "title": "Fix Database Timeout",
      "description": "Database locks up on heavy concurrent requests",
      "status": "open",
      "owner_id": 1,
      "user_id": 1,
      "created_at": "2026-07-16T14:45:00Z",
      "updated_at": "2026-07-16T14:45:00Z"
    }
  ]
  ```

---

### 6. Get Own Ticket by ID
* **Method**: `GET`
* **Path**: `/tickets/{id}`
* **Auth Required**: Yes (Bearer Token)
* **Headers**: `Authorization: Bearer <JWT_TOKEN>`
* **Success Response (200 OK)**:
  ```json
  {
    "id": 1,
    "title": "Fix Database Timeout",
    "description": "Database locks up on heavy concurrent requests",
    "status": "open",
    "owner_id": 1,
    "user_id": 1,
    "created_at": "2026-07-16T14:45:00Z",
    "updated_at": "2026-07-16T14:45:00Z"
  }
  ```
* **Error Response (403 Forbidden)**: Attempting to access another user's ticket.
* **Error Response (404 Not Found)**: Ticket does not exist.

---

### 7. Update Own Ticket Status
* **Method**: `PATCH`
* **Path**: `/tickets/{id}/status`
* **Auth Required**: Yes (Bearer Token)
* **Headers**: `Authorization: Bearer <JWT_TOKEN>`
* **Request Body**:
  ```json
  {
    "status": "in_progress"
  }
  ```
* **Success Response (200 OK)**:
  ```json
  {
    "id": 1,
    "title": "Fix Database Timeout",
    "description": "Database locks up on heavy concurrent requests",
    "status": "in_progress",
    "owner_id": 1,
    "user_id": 1,
    "created_at": "2026-07-16T14:45:00Z",
    "updated_at": "2026-07-16T14:48:32Z"
  }
  ```
* **Error Response (400 Bad Request)**: Invalid status value or skipped/invalid transition (e.g. `open` -> `closed` directly, or reopening a `closed` ticket).
* **Error Response (403 Forbidden)**: Trying to update a ticket owned by another user.
* **Error Response (404 Not Found)**: Ticket does not exist.

---

## Deployment Instructions

### Deploy to Render (Web Services)
1. **Create Render Account**: Sign up at [render.com](https://render.com/).
2. **New Web Service**: Click **New +** -> **Web Service**.
3. **Connect Repository**: Connect your Git repository containing this code.
4. **Environment Settings**:
   - **Runtime**: Select `Docker`.
   - **Branch**: Select your main/master branch.
5. **Environment Variables**: Add the following variable in the Render dashboard configuration:
   - `JWT_SECRET`: A secure random string.
   - *(Note: Port is auto-injected by Render, and DB_PATH defaults to a path that Render supports, but you can also attach a persistent disk volume to `/data` in Render settings to persist SQLite between service deploys).*
6. **Deploy**: Render will automatically build the `Dockerfile` and start the service.
7. **Verify**: Use the generated live URL to call:
   ```bash
   curl https://your-service-name.onrender.com/health
   ```

### Deploy to Railway
1. **Create Railway Account**: Sign up at [railway.app](https://railway.app/).
2. **New Project**: Click **New Project** -> **Deploy from GitHub repo**.
3. **Variables**: Set environment variables in the variables tab:
   - `JWT_SECRET`
4. **Deploy**: Railway automatically detects the `Dockerfile`, compiles, and spins up the container.

---

## Deployed URL Reference
- **Public Service URL**: `https://ticket-system-service-demo.up.railway.app` *(Placeholder)*
- **Public Health Check**: `https://ticket-system-service-demo.up.railway.app/health` *(Placeholder)*

---

## Assumptions & Design Choices
- **Pure-Go SQLite**: Chosen `modernc.org/sqlite` instead of `go-sqlite3` to avoid any CGO compiling dependencies. This ensures compilation succeeds in container environments without needing cross-compilers or dynamic C-linking.
- **Ownership Conflict Resolution**: If a user attempts to access or modify a ticket belonging to someone else, the application consistently returns `403 Forbidden`. If the ticket does not exist at all, the application returns `404 Not Found`.
- **Dual ID Fields**: The Ticket response payload includes both `owner_id` and `user_id` mapped to the owner's database ID to satisfy different test specifications without breaking compatibility.
- **Login Identifier Flexibility**: Login requests accept either `username` or `email` inside the JSON payload.
