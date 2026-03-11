package main

import (
	"also-wrote/internal/auth"
	"also-wrote/internal/db"
	"also-wrote/internal/mailer"
	"also-wrote/internal/ratelimit"
	"also-wrote/internal/tmdb"
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var tmdbClient *tmdb.Client
var templates *template.Template
var database *db.DB

// Rate limiters: login is strict (5 per 15 min), TMDB/API are moderate (60 per min)
var (
	loginLimiter   = ratelimit.NewLimiter(5, 15*time.Minute)
	generalLimiter = ratelimit.NewLimiter(60, time.Minute)
)

// emailRegex validates format: local@domain.tld (single dot in domain, TLD 2–6 letters)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\.[a-zA-Z]{2,6}$`)

const maxRequestBodyBytes = 1 << 20 // 1 MB — cap request bodies to avoid DoS

func init() {
	loadEnv()
	conn := os.Getenv("DATABASE_URL")
	if conn == "" {
		conn = "postgres://localhost/also_wrote?sslmode=disable"
	}
	var err error
	database, err = db.Open(conn)
	if err != nil {
		log.Fatalf("Database: %v (set DATABASE_URL for production)", err)
	}
	token := os.Getenv("TMDB_API_TOKEN")
	if token == "" {
		log.Println("Warning: TMDB_API_TOKEN environment variable is not set")
	}
	tmdbClient = tmdb.NewClient(token)
	funcMap := template.FuncMap{
		"initial": func(s string) string {
			s = strings.TrimSpace(s)
			if s == "" {
				return "?"
			}
			r := []rune(s)
			return strings.ToUpper(string(r[0:1]))
		},
		"avatarColor": func(s string) string {
			colors := []string{"bg-purple-500", "bg-fuchsia-500"}
			h := 0
			for _, c := range s {
				h += int(c)
			}
			if h < 0 {
				h = -h
			}
			return colors[h%len(colors)]
		},
		"formatDate": func(s string) string {
			if s == "" {
				return s
			}
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return s
			}
			return t.Format("Jan 2 2006")
		},
	}
	templates, err = template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
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

func getCurrentUser(r *http.Request) *db.User {
	c, err := r.Cookie(auth.CookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	secret := auth.Secret()
	if secret == "" {
		return nil
	}
	userID, _ := auth.VerifyCookie(c.Value, secret)
	if userID == 0 {
		return nil
	}
	u, err := database.UserByID(userID)
	if err != nil || u == nil {
		return nil
	}
	return u
}

func main() {
	// Order doesn't matter because longest path match is used
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/search", ratelimit.Middleware(generalLimiter, handleSearch))
	http.HandleFunc("/show", ratelimit.Middleware(generalLimiter, handleShow))
	http.HandleFunc("/writer", ratelimit.Middleware(generalLimiter, handlePerson))
	http.HandleFunc("/episode", ratelimit.Middleware(generalLimiter, handleEpisode))
	// Auth & Favorite Writers (login: strict limit on POST only)
	http.HandleFunc("/login", ratelimit.MiddlewarePost(loginLimiter, handleLogin))
	http.HandleFunc("/auth/verify", handleAuthVerify)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/favorite-writers", ratelimit.Middleware(generalLimiter, handleFavoriteWriters))
	http.HandleFunc("/api/favorite-writers", ratelimit.Middleware(generalLimiter, handleFavoriteWritersAPI))
	http.HandleFunc("/api/favorite-writers/overlap-graph", ratelimit.Middleware(generalLimiter, handleFavoriteWritersOverlapGraph))
	http.HandleFunc("/api/favorite-writers/", ratelimit.Middleware(generalLimiter, handleFavoriteWritersAPIDelete))

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
		renderTemplate(w, r, "404.html", nil)
		return
	}
	user := getCurrentUser(r)

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

	renderTemplate(w, r, "index.html", map[string]interface{}{
		"User":           user,
		"SuggestedShows": suggestedShows,
	})
}

func renderError(w http.ResponseWriter, r *http.Request, title, message string, status int) {
	w.WriteHeader(status)
	renderTemplate(w, r, "404.html", map[string]interface{}{
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
		renderError(w, r, "Search Error", "Error searching shows: "+err.Error(), http.StatusInternalServerError)
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
		renderTemplate(w, r, "search_results.html", map[string]interface{}{
			"Query":   query,
			"Results": results,
		})
		return
	}

	// If no TV shows are found, search writers
	people, err := tmdbClient.SearchPeople(query)
	if err != nil {
		renderError(w, r, "Search Error", "Error searching people: "+err.Error(), http.StatusInternalServerError)
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
			http.Redirect(w, r, fmt.Sprintf("/writer?id=%d", peopleResults[0].ID), http.StatusFound)
			return
		}
		renderTemplate(w, r, "search_results_people.html", map[string]interface{}{
			"Query":   query,
			"Results": peopleResults,
		})
		return
	}

	renderTemplate(w, r, "no_results.html", map[string]interface{}{
		"Query": query,
	})
}

func handleShow(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		renderError(w, r, "Invalid Show", "The show ID provided is invalid.", http.StatusBadRequest)
		return
	}

	showDetails, err := tmdbClient.GetTVShowDetails(id)
	if err != nil {
		renderError(w, r, "Show Not Found", "We couldn't find details for this show. It may not exist or there was a problem fetching the data.", http.StatusNotFound)
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

	renderTemplate(w, r, "show_details.html", map[string]interface{}{
		"Show":    showDetails,
		"Seasons": allSeasons,
	})
}

func handlePerson(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		renderError(w, r, "Invalid Person", "The person ID provided is invalid.", http.StatusBadRequest)
		return
	}

	person, err := tmdbClient.GetPersonDetails(id)
	if err != nil {
		renderError(w, r, "Person Not Found", "We couldn't find details for this person.", http.StatusNotFound)
		return
	}

	credits, err := tmdbClient.GetPersonTVCredits(id)
	if err != nil {
		renderError(w, r, "Credits Not Found", "We couldn't fetch credits for this person.", http.StatusInternalServerError)
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

	user := getCurrentUser(r)
	var isFavorited bool
	if user != nil {
		isFavorited, _ = database.IsFavoriteWriter(user.ID, id)
	}
	renderTemplate(w, r, "person_details.html", map[string]interface{}{
		"User":         user,
		"Person":       person,
		"Credits":      writingCredits,
		"IsFavorited":  isFavorited,
	})
}

func getOrCreateCSRFToken(w http.ResponseWriter, r *http.Request) string {
	token := auth.TokenFromRequest(r)
	if auth.ValidToken(token) {
		return token
	}
	token, err := auth.GenerateToken()
	if err != nil {
		return ""
	}
	auth.SetCookie(w, token)
	return token
}

func renderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	if _, ok := data["User"]; !ok {
		data["User"] = getCurrentUser(r)
	}
	if _, ok := data["CsrfToken"]; !ok {
		data["CsrfToken"] = getOrCreateCSRFToken(w, r)
	}
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
		renderError(w, r, "Invalid Show ID", "The show ID provided is invalid.", http.StatusBadRequest)
		return
	}
	seasonNum, err := strconv.Atoi(seasonNumStr)
	if err != nil {
		renderError(w, r, "Invalid Season Number", "The season number provided is invalid.", http.StatusBadRequest)
		return
	}
	episodeNum, err := strconv.Atoi(episodeNumStr)
	if err != nil {
		renderError(w, r, "Invalid Episode Number", "The episode number provided is invalid.", http.StatusBadRequest)
		return
	}

	episode, err := tmdbClient.GetEpisodeDetails(showID, seasonNum, episodeNum)
	if err != nil {
		renderError(w, r, "Episode Not Found", "We couldn't find details for this episode.", http.StatusNotFound)
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

	user := getCurrentUser(r)
	renderTemplate(w, r, "episode_details.html", map[string]interface{}{
		"User":         user,
		"Episode":      episode,
		"Show":         show,
		"WritingStaff": writingStaff,
	})
}

// --- Auth & Favorite Writers ---

func handleLogin(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		if !auth.VerifyCSRF(r, r.Form.Get("csrf_token")) {
			renderTemplate(w, r, "login.html", map[string]interface{}{
				"User":  nil,
				"Error": "Invalid request. Please try again.",
			})
			return
		}
		email := strings.TrimSpace(r.Form.Get("email"))
		if email == "" {
			renderTemplate(w, r, "login.html", map[string]interface{}{
				"User":  nil,
				"Error": "Please enter your email.",
			})
			return
		}
		if len(email) > 254 {
			renderTemplate(w, r, "login.html", map[string]interface{}{
				"User":  nil,
				"Error": "Please enter a valid email address.",
			})
			return
		}
		if !emailRegex.MatchString(email) {
			renderTemplate(w, r, "login.html", map[string]interface{}{
				"User":  nil,
				"Error": "Please enter a valid email address.",
			})
			return
		}
		raw, tokenHash, err := auth.NewMagicLinkToken()
		if err != nil {
			renderError(w, r, "Error", "Could not create sign-in link.", http.StatusInternalServerError)
			return
		}
		expiresAt := time.Now().Add(auth.TokenExpiry)
		if err := database.SaveMagicLinkToken(tokenHash, email, expiresAt); err != nil {
			log.Printf("SaveMagicLinkToken: %v", err)
			renderError(w, r, "Error", "Could not create sign-in link.", http.StatusInternalServerError)
			return
		}
		baseURL := os.Getenv("APP_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}
		link := mailer.MagicLinkURL(baseURL, auth.RawTokenToURLParam(raw))
		if err := mailer.SendMagicLink(email, link); err != nil {
			log.Printf("SendMagicLink: %v", err)
			renderError(w, r, "Error", "Could not send email. Try again or check server logs for the link.", http.StatusInternalServerError)
			return
		}
		renderTemplate(w, r, "check_email.html", map[string]interface{}{
			"User":  nil,
			"Email": email,
		})
		return
	}
	errMsg := r.URL.Query().Get("error")
	var errInterface interface{}
	if errMsg != "" {
		switch errMsg {
		case "missing", "invalid":
			errInterface = "Invalid or missing sign-in link. Request a new one below."
		case "expired":
			errInterface = "That link has expired. Request a new one below."
		default:
			errInterface = "Something went wrong. Please try again."
		}
	}
	renderTemplate(w, r, "login.html", map[string]interface{}{
		"User":  nil,
		"Error": errInterface,
	})
}

func handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	tokenParam := r.URL.Query().Get("token")
	if tokenParam == "" {
		http.Redirect(w, r, "/login?error=missing", http.StatusFound)
		return
	}
	raw, err := auth.URLParamToRaw(tokenParam)
	if err != nil {
		http.Redirect(w, r, "/login?error=invalid", http.StatusFound)
		return
	}
	tokenHash := auth.TokenHash(raw)
	email, err := database.ConsumeMagicLinkToken(tokenHash)
	if err != nil || email == "" {
		http.Redirect(w, r, "/login?error=expired", http.StatusFound)
		return
	}
	user, err := database.GetOrCreateUser(email)
	if err != nil {
		log.Printf("GetOrCreateUser: %v", err)
		http.Redirect(w, r, "/login?error=server", http.StatusFound)
		return
	}
	secret := auth.Secret()
	if secret == "" {
		log.Println("SESSION_SECRET not set; session will not be stored")
	} else {
		auth.SetSession(w, user, secret)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if !auth.VerifyCSRF(r, r.Form.Get("csrf_token")) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	auth.ClearSession(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func handleFavoriteWriters(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	personIDs, err := database.FavoriteWriterPersonIDs(user.ID)
	if err != nil {
		renderError(w, r, "Error", "Could not load your favorite writers.", http.StatusInternalServerError)
		return
	}
	var writers []*tmdb.Person
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, id := range personIDs {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			p, err := tmdbClient.GetPersonDetails(pid)
			if err == nil && p != nil {
				mu.Lock()
				writers = append(writers, p)
				mu.Unlock()
			}
		}(id)
	}
	wg.Wait()
	renderTemplate(w, r, "favorite_writers.html", map[string]interface{}{
		"User":    user,
		"Writers": writers,
	})
}

func handleFavoriteWritersAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !auth.VerifyCSRF(r, r.Header.Get("X-CSRF-Token")) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var body struct {
		PersonID int `json:"person_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PersonID == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if err := database.AddFavoriteWriter(user.ID, body.PersonID); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func handleFavoriteWritersOverlapGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	personIDs, err := database.FavoriteWriterPersonIDs(user.ID)
	if err != nil {
		http.Error(w, "Could not load favorite writers", http.StatusInternalServerError)
		return
	}
	if len(personIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		})
		return
	}

	type graphNode struct {
		ID           string  `json:"id"`
		Type         string  `json:"type"` // "writer" or "show"
		Name         string  `json:"name"`
		PosterPath   string  `json:"poster_path,omitempty"`
		ProfilePath  string  `json:"profile_path,omitempty"`
		Priority     float64 `json:"priority,omitempty"`
		WriterCount  int     `json:"writer_count,omitempty"`  // for shows: number of favorite writers who worked on it
		FirstAirDate string  `json:"first_air_date,omitempty"` // for shows: YYYY-MM-DD
	}
	type graphEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}

	// Fetch writer details (name, profile) and build writer nodes
	writerNodes := make(map[int]*graphNode)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, pid := range personIDs {
		wg.Add(1)
		go func(personID int) {
			defer wg.Done()
			p, err := tmdbClient.GetPersonDetails(personID)
			if err != nil || p == nil {
				return
			}
			mu.Lock()
			writerNodes[personID] = &graphNode{
				ID:          fmt.Sprintf("w-%d", personID),
				Type:        "writer",
				Name:        p.Name,
				ProfilePath: p.ProfilePath,
			}
			mu.Unlock()
		}(pid)
	}
	wg.Wait()

	// showID -> show node info and writer list for priority
	shows := make(map[int]*graphNode)
	writerToShows := make(map[int][]int) // personID -> showIDs
	for _, pid := range personIDs {
		wg.Add(1)
		go func(personID int) {
			defer wg.Done()
			credits, err := tmdbClient.GetPersonTVCredits(personID)
			if err != nil || credits == nil {
				return
			}
			var showIDs []int
			for _, c := range credits.Crew {
				if c.Department != "Writing" {
					continue
				}
				mu.Lock()
				s, ok := shows[c.ID]
				if !ok {
					s = &graphNode{
						ID:           fmt.Sprintf("s-%d", c.ID),
						Type:         "show",
						Name:         c.Name,
						PosterPath:   c.PosterPath,
						FirstAirDate: c.FirstAirDate,
					}
					shows[c.ID] = s
				}
				s.Priority += float64(c.EpisodeCount)
				showIDs = append(showIDs, c.ID)
				mu.Unlock()
			}
			mu.Lock()
			writerToShows[personID] = showIDs
			mu.Unlock()
		}(pid)
	}
	wg.Wait()

	// Count distinct writers per show for priority
	showWriterCount := make(map[int]int)
	for _, showIDs := range writerToShows {
		seen := make(map[int]bool)
		for _, sid := range showIDs {
			if !seen[sid] {
				seen[sid] = true
				showWriterCount[sid]++
			}
		}
	}
	for sid, s := range shows {
		c := showWriterCount[sid]
		s.WriterCount = c
		s.Priority = float64(c)*50 + s.Priority
	}

	// Build edges: writer -> show for each credit
	var edges []graphEdge
	for personID, showIDs := range writerToShows {
		writerID := fmt.Sprintf("w-%d", personID)
		for _, sid := range showIDs {
			edges = append(edges, graphEdge{Source: writerID, Target: fmt.Sprintf("s-%d", sid)})
		}
	}

	// Collect all nodes (writers first, then shows)
	nodes := make([]*graphNode, 0, len(writerNodes)+len(shows))
	for _, pid := range personIDs {
		if n := writerNodes[pid]; n != nil {
			nodes = append(nodes, n)
		}
	}
	for _, s := range shows {
		nodes = append(nodes, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

func handleFavoriteWritersAPIDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !auth.VerifyCSRF(r, r.Header.Get("X-CSRF-Token")) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/favorite-writers/")
	personID, err := strconv.Atoi(strings.Trim(path, "/"))
	if err != nil || personID <= 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if err := database.RemoveFavoriteWriter(user.ID, personID); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}
