# Writer Fan

A simple, elegant Go application to discover the writers behind your favorite TV shows using the TMDB API.

## Features

- **Search by Show**: Find TV series by title.
- **Episode Details**: Browse all episodes organized by season.
- **Writer Credits**: Identify writers for each episode.
- **Writer Profiles**: Click on a writer to see their other works.

## Setup

1.  **Clone the repository** (if you haven't already).
2.  **Run the application**:
    ```bash
    go run main.go
    ```
3.  **Open your browser**:
    Visit [http://localhost:8080](http://localhost:8080)

## Project Structure

- `main.go`: Main application entry point and HTTP server.
- `internal/tmdb`: TMDB API client implementation.
- `templates/`: HTML templates for the UI.
- `static/`: Static assets (if any).

## Dependencies

- Go 1.16+
- Tailwind CSS (via CDN)
