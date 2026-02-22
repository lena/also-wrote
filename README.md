# Also Wrote

A simple Go application to discover the writers behind your favorite TV shows using the TMDB API.

## **Live Demo:**

<a href="https://also-wrote.onrender.com" target="_blank" rel="noopener noreferrer">https://also-wrote.onrender.com</a>
<br>

## Features

- **Search by Show**: Find TV series by title.
- **Episode Details**: Browse all episodes organized by season, with full writing staff lists.
- **Writer Credits**: Identify writers for each episode.
- **Writer Profiles**: Click on a writer to see their other works and specific credited episodes.

## Screenshots

<div align="center">
  <img src="images/landing_page.png" alt="Landing Page" width="500">
  <br><em>Landing Page</em><br><br>

  <img src="images/show_details_page.png" alt="Show Details" width="500">
  <br><em>Show Details</em><br><br>

  <img src="images/writer_details_page.png" alt="Writer Details" width="500">
  <br><em>Writer Details</em><br><br>

  <img src="images/episode_details_page.png" alt="Episode Details" width="500">
  <br><em>Episode Details</em>
</div>

## Setup

1.  **Clone the repository** (if you haven't already).
2.  **Create .env file**:
    Copy `.env.example` to `.env` and add your [TMDB API Token](https://www.themoviedb.org/settings/api).
    ```bash
    cp .env.example .env
    ```
3.  **Run the application**:
    ```bash
    go run main.go
    ```
4.  **Open your browser**:
    Visit [http://localhost:8080](http://localhost:8080)

## Project Structure

- `main.go`: Main application entry point and HTTP server.
- `internal/tmdb`: TMDB API client implementation.
- `templates/`: HTML templates for the UI.
- `static/`: Static assets (if any).

## Dependencies

- Go 1.16+
- Tailwind CSS (via CDN)
