package main

import (
	"also-wrote/internal/tmdb"
	"bufio"
	"fmt"
	"html/template"
	"log"
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
	http.HandleFunc("/episode", handleEpisode)

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

	suggestedTitles := []string{"The Wire", "Better Call Saul", "BoJack Horseman"}
	var suggestedShows []tmdb.TVShow
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, title := range suggestedTitles {
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

	// Create a map for quick lookup by lowercased name
	showMap := make(map[string]tmdb.TVShow)
	for _, s := range suggestedShows {
		showMap[strings.ToLower(s.Name)] = s
	}

	orderedShows := make([]tmdb.TVShow, 0, len(suggestedTitles))
	seenIDs := make(map[int]bool)

	for _, title := range suggestedTitles {
		lowerTitle := strings.ToLower(title)

		var matchedShow tmdb.TVShow
		found := false

		// 1. Exact match in map
		if s, ok := showMap[lowerTitle]; ok {
			matchedShow = s
			found = true
		} else {
			// 2. Contains match
			for _, s := range suggestedShows {
				if strings.Contains(strings.ToLower(s.Name), lowerTitle) {
					matchedShow = s
					found = true
					break
				}
			}
		}

		if found && !seenIDs[matchedShow.ID] {
			orderedShows = append(orderedShows, matchedShow)
			seenIDs[matchedShow.ID] = true
		}
	}

	renderTemplate(w, "index.html", map[string]interface{}{
		"SuggestedShows": orderedShows,
	})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// 1. Search TV Shows
	results, err := tmdbClient.SearchTVShows(query)
	if err != nil {
		http.Error(w, "Error searching shows: "+err.Error(), http.StatusInternalServerError)
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

	// 2. Search People
	people, err := tmdbClient.SearchPeople(query)
	if err != nil {
		http.Error(w, "Error searching people: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var writers []tmdb.Person
	for _, p := range people {
		if p.KnownForDepartment == "Writing" {
			writers = append(writers, p)
		}
	}

	if len(writers) > 0 {
		if len(writers) == 1 {
			http.Redirect(w, r, fmt.Sprintf("/person?id=%d", writers[0].ID), http.StatusFound)
			return
		}
		renderTemplate(w, "search_results_people.html", map[string]interface{}{
			"Query":   query,
			"Results": writers,
		})
		return
	}

	// 3. No results found
	renderTemplate(w, "no_results.html", query)
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
	var wg sync.WaitGroup
	resultsCh := make(chan *tmdb.Season, len(showDetails.Seasons))
	semaphore := make(chan struct{}, 10) // Limit concurrency

	for _, s := range showDetails.Seasons {
		if s.SeasonNumber == 0 {
			continue
		} // Skip specials usually

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
		allSeasons = append(allSeasons, SeasonWithEpisodes{*s})
	}

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

	type WriterCredit struct {
		tmdb.Credit
		Episodes []tmdb.Episode
	}

	var writingCredits []WriterCredit
	var wg sync.WaitGroup
	// Estimate capacity, though channel can grow if buffered enough or consumed async
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

	// Sort by FirstAirDate descending (newest first)
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
		http.Error(w, "Invalid show ID", http.StatusBadRequest)
		return
	}
	seasonNum, err := strconv.Atoi(seasonNumStr)
	if err != nil {
		http.Error(w, "Invalid season number", http.StatusBadRequest)
		return
	}
	episodeNum, err := strconv.Atoi(episodeNumStr)
	if err != nil {
		http.Error(w, "Invalid episode number", http.StatusBadRequest)
		return
	}

	episode, err := tmdbClient.GetEpisodeDetails(showID, seasonNum, episodeNum)
	if err != nil {
		http.Error(w, "Error fetching episode details: "+err.Error(), http.StatusInternalServerError)
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
			if credit.Department == "Writing" {
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
