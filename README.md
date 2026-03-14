# Also Wrote

A simple Go application to discover the writers behind your favorite TV shows using the TMDB API.

## **Live Demo:**

<a href="https://also-wrote.onrender.com" target="_blank" rel="noopener noreferrer">https://also-wrote.onrender.com</a>
<br>

## Features

- **Search by Show**: Find TV series by title.
- **Search by Writer**: Find TV writers by name.
- **Episode Details**: Browse all episodes organized by season, with full writing staff lists.
- **Writer Credits**: Identify writers for each episode.
- **Writer Profiles**: Click on a writer to see their other works and specific credited episodes.
- **Sign in with Magic Link**: Enter your email to receive a one-time sign-in link (no password).
- **Favorite Writers**: Save your favorite writers.

## Screenshots

<div align="center">
  <img src="images/landing_page.png" alt="Landing Page" width="500">
  <br><em>Landing Page</em><br><br>

  <img src="images/show_details_page.png" alt="Show Details" width="500">
  <br><em>Show Details</em><br><br>

  <img src="images/writer_details_page.png" alt="Writer Details" width="500">
  <br><em>Writer Details</em><br><br>

  <img src="images/episode_details_page.png" alt="Episode Details" width="500">
  <br><em>Episode Details</em><br><br>

  ### Logged in features
  <img src="images/favorite_writers_page.png" alt="Favorite Writers" width="500">
  <br><em>Favorite Writers Page</em><br><br>

  <img src="images/favorite_writers_overlap_modal.png" alt="Favorite Writers Overlap Modal" width="500">
  <br><em>Writer/Series Overlap Graph</em><br><br>

  <img src="images/favorite_writers_overlap_modal_filtered.png" alt="Favorite Writers Overlap Modal (filtered)" width="500">
  <br><em>Writer/Series Overlap Graph (filtered)</em><br><br>
</div>

## Setup

1.  **Clone the repository** (if you haven't already).
2.  **PostgreSQL**: Create a local database for development:
    ```bash
    createdb also_wrote
    ```
3.  **Create .env file**:
    Copy `.env.example` to `.env` and add your [TMDB API Token](https://www.themoviedb.org/settings/api) and your username to the `DATABASE_URL`.
    Optionally set SMTP vars if you want magic-link emails sent instead of printed in the server log.
    ```bash
    cp .env.example .env
    ```
4.  **Build the frontend** (TypeScript/React):
    ```bash
    cd frontend && npm install && npm run build && cd ..
    ```
    Requires Node 20.19+ or 22.12+ for the Vite build.
5.  **Run the application**:
    ```bash
    go run main.go
    ```
6.  **Open your browser**:
    Visit [http://localhost:8080](http://localhost:8080). 
    Without SMTP configured, magic links are printed in the server log when you request a sign-in email.

**Development (frontend hot reload)**  
Run the Go server and the Vite dev server in two terminals. The frontend proxies `/api`, `/auth`, and `/static` to the Go server.
```bash
# Terminal 1
go run main.go

# Terminal 2
cd frontend && npm run dev
```
Then open [http://localhost:5173](http://localhost:5173). After changes, build the frontend once before deploying.

## Project Structure

- `main.go`: Main application entry point, HTTP server, and JSON API for the React app.
- `internal/tmdb`: TMDB API client implementation.
- `internal/auth`, `internal/db`, `internal/mailer`, `internal/ratelimit`: Backend packages.
- `frontend/`: TypeScript/React SPA (Vite, React Router, Tailwind). Built output in `frontend/dist` is served by the Go server in production.
- `static/`: Static assets (e.g. favicon).

## Dependencies

- Go 1.24+
- Node 20.19+ or 22.12+ (for frontend build)
- PostgreSQL
- Frontend: React, TypeScript, Vite, React Router, Tailwind CSS, D3 (overlap graph)

## Testing

**To run unit tests** (auth, ratelimit — no database or API keys required):

```bash
go test ./...
```

**To run DB integration tests** Use a separate test database so tests don’t touch dev data.

1.  **PostgreSQL**: Create a local database for tests:
    ```bash
    createdb also_wrote_test
    ```

2. Run tests with the `integration` build tag:
   ```bash
   go test -tags=integration ./...
   ```

DB tests use `TEST_DATABASE_URL` if set, otherwise `postgres://localhost/also_wrote_test?sslmode=disable`. They create and remove temporary users so the test database stays clean.

**Pre-commit hook** — Run all tests (unit + DB integration) before every commit:

```bash
cp scripts/pre-commit .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit
```
