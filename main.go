package main

import (
	"also-wrote/internal/tmdb"
	"bufio"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	// Declare error type explicitly because using assignment on next line
	// to update the the *package-level* templates variable
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
}

func loadEnv() {
	// Open .env file and load necessary TMDB API token
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
	// Order doesn't matter because longest path match is used
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/search", handleSearch)
	http.HandleFunc("/show", handleShow)
	http.HandleFunc("/person", handlePerson)
	http.HandleFunc("/episode", handleEpisode)

	// Serve static files (favicon)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		http.ServeFile(w, r, "static/favicon.svg")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		renderTemplate(w, "404.html", nil)
		return
	}

	// Pick 3 random titles from the list, then fetch their data in parallel using goroutines
	allSuggestedTitles := []string{
		"Arrested Development",
		"Atlanta",
		"Battlestar Galactica",
		"Better Call Saul",
		"Better Off Ted",
		"BoJack Horseman",
		"Buffy the Vampire Slayer",
		"Glow",
		"Hacks",
		"Insecure",
		"Parks and Recreation",
		"The Bear",
		"The Simpsons",
		"The Sopranos",
		"The Wire",
	}

	// Shuffle using Fisher-Yates (Knuth) shuffle algorithm on a copy of the slice
	suggestedTitles := make([]string, len(allSuggestedTitles))
	copy(suggestedTitles, allSuggestedTitles)
	rand.Shuffle(len(suggestedTitles), func(i, j int) {
		suggestedTitles[i], suggestedTitles[j] = suggestedTitles[j], suggestedTitles[i]
	})

	selectedTitles := suggestedTitles[:3]

	var suggestedShows []tmdb.TVShow
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, title := range selectedTitles {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			results, err := tmdbClient.SearchTVShows(t)
			if err == nil && len(results) > 0 {
				mu.Lock()
				suggestedShows = append(suggestedShows, results[0])
				mu.Unlock()
			}
		}(title)
	}
	wg.Wait()

	renderTemplate(w, "index.html", map[string]interface{}{
		"SuggestedShows": suggestedShows,
	})
}

func renderError(w http.ResponseWriter, title, message string, status int) {
	w.WriteHeader(status)
	renderTemplate(w, "404.html", map[string]interface{}{
		"ErrorTitle":   title,
		"ErrorMessage": message,
	})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	// The search logic handles both TV shows and writers.
	// Preference for TV shows since the point is writer discovery.
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// First, search TV shows
	results, err := tmdbClient.SearchTVShows(query)
	if err != nil {
		renderError(w, "Search Error", "Error searching shows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(results) > 0 {
		// Filter for exact matches (case-insensitive)
		var exactMatches []tmdb.TVShow
		for _, show := range results {
			if strings.EqualFold(show.Name, query) {
				exactMatches = append(exactMatches, show)
			}
		}

		// If we have one or more exact matches, prioritize them
		if len(exactMatches) > 0 {
			results = exactMatches
		}

		if len(results) == 1 {
			http.Redirect(w, r, fmt.Sprintf("/show?id=%d", results[0].ID), http.StatusFound)
			return
		}
		renderTemplate(w, "search_results.html", map[string]interface{}{
			"Query":   query,
			"Results": results,
		})
		return
	}

	// If no TV shows are found, search writers
	people, err := tmdbClient.SearchPeople(query)
	if err != nil {
		renderError(w, "Search Error", "Error searching people: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var writers []tmdb.Person
	for _, p := range people {
		if p.KnownForDepartment == "Writing" {
			writers = append(writers, p)
		}
	}

	// If we find any writers, display only writers
	// If we find other people who may be better known as actors or producers, display them
	var peopleResults []tmdb.Person
	if len(writers) > 0 {
		peopleResults = writers
	} else {
		peopleResults = people
	}

	if len(peopleResults) > 0 {
		if len(peopleResults) == 1 {
			http.Redirect(w, r, fmt.Sprintf("/person?id=%d", peopleResults[0].ID), http.StatusFound)
			return
		}
		renderTemplate(w, "search_results_people.html", map[string]interface{}{
			"Query":   query,
			"Results": peopleResults,
		})
		return
	}

	renderTemplate(w, "no_results.html", query)
}

func handleShow(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		renderError(w, "Invalid Show", "The show ID provided is invalid.", http.StatusBadRequest)
		return
	}

	showDetails, err := tmdbClient.GetTVShowDetails(id)
	if err != nil {
		renderError(w, "Show Not Found", "We couldn't find details for this show. It may not exist or there was a problem fetching the data.", http.StatusNotFound)
		return
	}

	var allSeasons []*tmdb.Season
	var wg sync.WaitGroup
	resultsCh := make(chan *tmdb.Season, len(showDetails.Seasons))
	semaphore := make(chan struct{}, 10) // Limit concurrency

	for _, s := range showDetails.Seasons {
		if s.SeasonNumber == 0 {
			continue
		} // Skip specials

		wg.Add(1)
		go func(sNum int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			seasonDetails, err := tmdbClient.GetSeasonDetails(id, sNum)
			if err != nil {
				log.Printf("Error fetching season %d: %v", sNum, err)
				return
			}
			resultsCh <- seasonDetails
		}(s.SeasonNumber)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for s := range resultsCh {
		allSeasons = append(allSeasons, s)
	}

	// Sort seasons by season number
	sort.Slice(allSeasons, func(i, j int) bool {
		return allSeasons[i].SeasonNumber < allSeasons[j].SeasonNumber
	})

	renderTemplate(w, "show_details.html", map[string]interface{}{
		"Show":    showDetails,
		"Seasons": allSeasons,
	})
}

func handlePerson(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		renderError(w, "Invalid Person", "The person ID provided is invalid.", http.StatusBadRequest)
		return
	}

	person, err := tmdbClient.GetPersonDetails(id)
	if err != nil {
		renderError(w, "Person Not Found", "We couldn't find details for this person.", http.StatusNotFound)
		return
	}

	credits, err := tmdbClient.GetPersonTVCredits(id)
	if err != nil {
		renderError(w, "Credits Not Found", "We couldn't fetch credits for this person.", http.StatusInternalServerError)
		return
	}

	type WriterCredit struct {
		tmdb.Credit
		Episodes []tmdb.Episode
	}

	var writingCredits []WriterCredit
	var wg sync.WaitGroup
	resultsCh := make(chan WriterCredit, len(credits.Crew))
	semaphore := make(chan struct{}, 10) // Limit concurrency

	for _, credit := range credits.Crew {
		if credit.Department == "Writing" {
			wg.Add(1)
			go func(c tmdb.Credit) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				var eps []tmdb.Episode
				if c.CreditID != "" {
					details, err := tmdbClient.GetCreditDetails(c.CreditID)
					if err == nil && details != nil {
						eps = details.Media.Episodes
					} else {
						log.Printf("Error fetching credit details for %s: %v", c.CreditID, err)
					}
				}

				resultsCh <- WriterCredit{
					Credit:   c,
					Episodes: eps,
				}
			}(credit)
		}
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for wc := range resultsCh {
		writingCredits = append(writingCredits, wc)
	}

	// Sort by show's first air date descending (newest first)
	sort.Slice(writingCredits, func(i, j int) bool {
		return writingCredits[i].FirstAirDate > writingCredits[j].FirstAirDate
	})

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

func handleEpisode(w http.ResponseWriter, r *http.Request) {
	showIDStr := r.URL.Query().Get("show_id")
	seasonNumStr := r.URL.Query().Get("season")
	episodeNumStr := r.URL.Query().Get("episode")

	showID, err := strconv.Atoi(showIDStr)
	if err != nil {
		renderError(w, "Invalid Show ID", "The show ID provided is invalid.", http.StatusBadRequest)
		return
	}
	seasonNum, err := strconv.Atoi(seasonNumStr)
	if err != nil {
		renderError(w, "Invalid Season Number", "The season number provided is invalid.", http.StatusBadRequest)
		return
	}
	episodeNum, err := strconv.Atoi(episodeNumStr)
	if err != nil {
		renderError(w, "Invalid Episode Number", "The episode number provided is invalid.", http.StatusBadRequest)
		return
	}

	episode, err := tmdbClient.GetEpisodeDetails(showID, seasonNum, episodeNum)
	if err != nil {
		renderError(w, "Episode Not Found", "We couldn't find details for this episode.", http.StatusNotFound)
		return
	}

	show, err := tmdbClient.GetTVShowDetails(showID)
	if err != nil {
		log.Printf("Error fetching show details: %v", err)
	}

	seasonCredits, err := tmdbClient.GetSeasonAggregateCredits(showID, seasonNum)
	var writingStaff []tmdb.AggregateCredit
	if err == nil {
		for _, credit := range seasonCredits.Crew {
			if credit.Department == "Writing" && credit.ID > 0 { // Filter out credits with ID 0
				writingStaff = append(writingStaff, credit)
			}
		}
	} else {
		log.Printf("Error fetching season credits: %v", err)
	}

	renderTemplate(w, "episode_details.html", map[string]interface{}{
		"Episode":      episode,
		"Show":         show,
		"WritingStaff": writingStaff,
	})
}
