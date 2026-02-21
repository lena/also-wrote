package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"writer-fan/internal/tmdb"
)

var tmdbClient *tmdb.Client
var templates *template.Template

func init() {
	loadEnv()
	token := os.Getenv("TMDB_API_TOKEN")
	if token == "" {
		log.Println("Warning: TMDB_API_TOKEN environment variable is not set")
	}
	tmdbClient = tmdb.NewClient(token)
	// Parse all templates in the templates folder
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
}

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/search", handleSearch)
	http.HandleFunc("/show", handleShow)
	http.HandleFunc("/person", handlePerson)

	// Serve static files (CSS, images)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "index.html", nil)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	results, err := tmdbClient.SearchTVShows(query)
	if err != nil {
		http.Error(w, "Error searching shows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		renderTemplate(w, "no_results.html", query)
		return
	}

	if len(results) == 1 {
		http.Redirect(w, r, fmt.Sprintf("/show?id=%d", results[0].ID), http.StatusFound)
		return
	}

	renderTemplate(w, "search_results.html", map[string]interface{}{
		"Query":   query,
		"Results": results,
	})
}

func handleShow(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid show ID", http.StatusBadRequest)
		return
	}

	showDetails, err := tmdbClient.GetTVShowDetails(id)
	if err != nil {
		http.Error(w, "Error fetching show details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	type SeasonWithEpisodes struct {
		tmdb.Season
	}

	var allSeasons []SeasonWithEpisodes
	for _, s := range showDetails.Seasons {
		if s.SeasonNumber == 0 {
			continue
		} // Skip specials usually
		seasonDetails, err := tmdbClient.GetSeasonDetails(id, s.SeasonNumber)
		if err != nil {
			log.Printf("Error fetching season %d: %v", s.SeasonNumber, err)
			continue
		}
		allSeasons = append(allSeasons, SeasonWithEpisodes{*seasonDetails})
	}

	renderTemplate(w, "show_details.html", map[string]interface{}{
		"Show":    showDetails,
		"Seasons": allSeasons,
	})
}

func handlePerson(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid person ID", http.StatusBadRequest)
		return
	}

	person, err := tmdbClient.GetPersonDetails(id)
	if err != nil {
		http.Error(w, "Error fetching person details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	credits, err := tmdbClient.GetPersonTVCredits(id)
	if err != nil {
		http.Error(w, "Error fetching person credits: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var writingCredits []tmdb.Credit
	for _, credit := range credits.Crew {
		if credit.Department == "Writing" {
			writingCredits = append(writingCredits, credit)
		}
	}

	renderTemplate(w, "person_details.html", map[string]interface{}{
		"Person":  person,
		"Credits": writingCredits,
	})
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
