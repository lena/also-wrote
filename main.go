package main

import (
	"also-wrote/internal/auth"
	"also-wrote/internal/db"
	"also-wrote/internal/mailer"
	"also-wrote/internal/ratelimit"
	"also-wrote/internal/tmdb"
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var tmdbClient *tmdb.Client
var database *db.DB

// SPA files are embedded so the app works regardless of working directory (e.g. on Render).
// Build with frontend/dist present: cd frontend && npm run build && cd .. && go build
//go:embed frontend/dist
var spaFS embed.FS

// spaDistRoot is frontend/dist as fs.FS, set in main after stripping the prefix
var spaDistRoot fs.FS

// diskSPARoot is "frontend/dist" for fallback when embed is empty (e.g. go run . without prior build)
var diskSPARoot string

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

func isAdmin(u *db.User) bool {
	if u == nil {
		return false
	}
	return u.Email == os.Getenv("ADMIN_EMAIL")
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
	// API and auth first (longest path match)
	http.HandleFunc("/auth/verify", handleAuthVerify)
	http.HandleFunc("/api/favorite-writers", ratelimit.Middleware(generalLimiter, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleAPIFavoriteWritersList(w, r)
			return
		}
		handleFavoriteWritersAPI(w, r)
	}))
	http.HandleFunc("/api/favorite-writers/overlap-graph", ratelimit.Middleware(generalLimiter, handleFavoriteWritersOverlapGraph))
	http.HandleFunc("/api/favorite-writers/", ratelimit.Middleware(generalLimiter, handleFavoriteWritersAPIDelete))

	// JSON API for React SPA
	http.HandleFunc("/api/me", ratelimit.Middleware(generalLimiter, handleAPIMe))
	http.HandleFunc("/api/home", ratelimit.Middleware(generalLimiter, handleAPIHome))
	http.HandleFunc("/api/search", ratelimit.Middleware(generalLimiter, handleAPISearch))
	http.HandleFunc("/api/show", ratelimit.Middleware(generalLimiter, handleAPIShow))
	http.HandleFunc("/api/writer", ratelimit.Middleware(generalLimiter, handleAPIWriter))
	http.HandleFunc("/api/episode", ratelimit.Middleware(generalLimiter, handleAPIEpisode))
	http.HandleFunc("/api/admin/users", ratelimit.Middleware(generalLimiter, handleAPIAdminUsers))
	http.HandleFunc("/api/login", ratelimit.Middleware(loginLimiter, handleAPILogin))
	http.HandleFunc("/api/logout", ratelimit.Middleware(generalLimiter, handleAPILogout))

	// Serve static files (favicon)
	staticFiles := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticFiles))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		http.ServeFile(w, r, "static/favicon.svg")
	})

	// SPA: serve embedded frontend/dist; fall back to disk for local dev
	spaDistRoot, _ = fs.Sub(spaFS, "frontend/dist")
	assetsRoot, _ := fs.Sub(spaDistRoot, "assets")
	diskSPARoot = resolveDiskSPARoot()
	assetsDir := filepath.Join(diskSPARoot, "assets")
	http.Handle("/assets/", http.StripPrefix("/assets/", spaOrDiskAssetsHandler(assetsRoot, assetsDir)))
	http.HandleFunc("/", handleSPA)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
	if err := database.RecordLogin(user.ID); err != nil {
		log.Printf("RecordLogin: %v", err)
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
	targetUserID := user.ID
	if uidStr := r.URL.Query().Get("user_id"); uidStr != "" {
		if !isAdmin(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil || uid <= 0 {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		targetUserID = uid
	}
	personIDs, err := database.FavoriteWriterPersonIDs(targetUserID)
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

// --- JSON API for React SPA ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func handleAPIMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := getCurrentUser(r)
	token := getOrCreateCSRFToken(w, r)
	type resp struct {
		User      *db.User `json:"user"`
		CsrfToken string   `json:"csrf_token"`
		Admin     bool     `json:"admin"`
	}
	writeJSON(w, http.StatusOK, resp{
		User:      user,
		CsrfToken: token,
		Admin:     isAdmin(user),
	})
}

func handleAPIHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	allSuggestedTitles := []string{
		"Arrested Development", "Atlanta", "Battlestar Galactica", "Better Call Saul", "Better Off Ted",
		"BoJack Horseman", "Buffy the Vampire Slayer", "Glow", "Hacks", "Insecure", "Parks and Recreation",
		"The Bear", "The Simpsons", "The Sopranos", "The Wire",
	}
	suggestedTitles := make([]string, len(allSuggestedTitles))
	copy(suggestedTitles, allSuggestedTitles)
	rand.Shuffle(len(suggestedTitles), func(i, j int) { suggestedTitles[i], suggestedTitles[j] = suggestedTitles[j], suggestedTitles[i] })
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"suggested_shows": suggestedShows})
}

func handleAPISearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing query"})
		return
	}
	results, err := tmdbClient.SearchTVShows(query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(results) > 0 {
		var exactMatches []tmdb.TVShow
		for _, show := range results {
			if strings.EqualFold(show.Name, query) {
				exactMatches = append(exactMatches, show)
			}
		}
		if len(exactMatches) > 0 {
			results = exactMatches
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"results_type": "shows",
			"query":        query,
			"shows":        results,
		})
		return
	}
	people, err := tmdbClient.SearchPeople(query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var writers []tmdb.Person
	for _, p := range people {
		if p.KnownForDepartment == "Writing" {
			writers = append(writers, p)
		}
	}
	var peopleResults []tmdb.Person
	if len(writers) > 0 {
		peopleResults = writers
	} else {
		peopleResults = people
	}
	if len(peopleResults) > 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"results_type": "people",
			"query":        query,
			"people":       peopleResults,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results_type": "none",
		"query":        query,
	})
}

func handleAPIShow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	showDetails, err := tmdbClient.GetTVShowDetails(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "show not found"})
		return
	}
	var allSeasons []*tmdb.Season
	var wg sync.WaitGroup
	resultsCh := make(chan *tmdb.Season, len(showDetails.Seasons))
	semaphore := make(chan struct{}, 10)
	for _, s := range showDetails.Seasons {
		if s.SeasonNumber == 0 {
			continue
		}
		wg.Add(1)
		go func(sNum int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			seasonDetails, err := tmdbClient.GetSeasonDetails(id, sNum)
			if err != nil {
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
	sort.Slice(allSeasons, func(i, j int) bool {
		return allSeasons[i].SeasonNumber < allSeasons[j].SeasonNumber
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"show":    showDetails,
		"seasons": allSeasons,
	})
}

type writerCreditAPI struct {
	tmdb.Credit
	Episodes []tmdb.Episode `json:"episodes"`
}

func handleAPIWriter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	person, err := tmdbClient.GetPersonDetails(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "person not found"})
		return
	}
	credits, err := tmdbClient.GetPersonTVCredits(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "credits not found"})
		return
	}
	var writingCredits []writerCreditAPI
	var wg sync.WaitGroup
	resultsCh := make(chan writerCreditAPI, len(credits.Crew))
	semaphore := make(chan struct{}, 10)
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
					}
				}
				resultsCh <- writerCreditAPI{Credit: c, Episodes: eps}
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
	sort.Slice(writingCredits, func(i, j int) bool {
		return writingCredits[i].FirstAirDate > writingCredits[j].FirstAirDate
	})
	user := getCurrentUser(r)
	var isFavorited bool
	if user != nil {
		isFavorited, _ = database.IsFavoriteWriter(user.ID, id)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"person":        person,
		"credits":       writingCredits,
		"is_favorited": isFavorited,
	})
}

func handleAPIEpisode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	showID, _ := strconv.Atoi(r.URL.Query().Get("show_id"))
	seasonNum, _ := strconv.Atoi(r.URL.Query().Get("season"))
	episodeNum, _ := strconv.Atoi(r.URL.Query().Get("episode"))
	if showID == 0 || seasonNum == 0 || episodeNum == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid show_id, season, or episode"})
		return
	}
	episode, err := tmdbClient.GetEpisodeDetails(showID, seasonNum, episodeNum)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "episode not found"})
		return
	}
	show, _ := tmdbClient.GetTVShowDetails(showID)
	seasonCredits, err := tmdbClient.GetSeasonAggregateCredits(showID, seasonNum)
	var writingStaff []tmdb.AggregateCredit
	if err == nil {
		for _, credit := range seasonCredits.Crew {
			if credit.Department == "Writing" && credit.ID > 0 {
				writingStaff = append(writingStaff, credit)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"episode":       episode,
		"show":          show,
		"writing_staff": writingStaff,
	})
}

func handleAPIFavoriteWritersList(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load favorite writers"})
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"writers": writers})
}

func handleAPIAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := getCurrentUser(r)
	if !isAdmin(user) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	list, err := database.ListUsersWithFavoriteCount()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"users": list})
}

func handleAPILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := getCurrentUser(r)
	if user != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	email := strings.TrimSpace(body.Email)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please enter your email."})
		return
	}
	if len(email) > 254 || !emailRegex.MatchString(email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please enter a valid email address."})
		return
	}
	raw, tokenHash, err := auth.NewMagicLinkToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not create sign-in link."})
		return
	}
	expiresAt := time.Now().Add(auth.TokenExpiry)
	if err := database.SaveMagicLinkToken(tokenHash, email, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not create sign-in link."})
		return
	}
	baseURL := os.Getenv("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	link := mailer.MagicLinkURL(baseURL, auth.RawTokenToURLParam(raw))
	if err := mailer.SendMagicLink(email, link); err != nil {
		log.Printf("SendMagicLink: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not send email. Try again later."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func handleAPILogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !auth.VerifyCSRF(r, r.Header.Get("X-CSRF-Token")) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	auth.ClearSession(w)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// resolveDiskSPARoot returns a path to frontend/dist where index.html exists (cwd or exe dir), or "frontend/dist" as fallback.
func resolveDiskSPARoot() string {
	indexName := filepath.Join("frontend", "dist", "index.html")
	if wd, err := os.Getwd(); err == nil {
		if p := filepath.Join(wd, indexName); pathExists(p) {
			return filepath.Join(wd, "frontend", "dist")
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if p := filepath.Join(dir, indexName); pathExists(p) {
			return filepath.Join(dir, "frontend", "dist")
		}
	}
	return filepath.Join(".", "frontend", "dist")
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func handleSPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	indexPath := filepath.Join(diskSPARoot, "index.html")
	if pathExists(indexPath) {
		http.ServeFile(w, r, indexPath)
		return
	}
	if f, err := spaDistRoot.Open("index.html"); err == nil {
		f.Close()
		http.ServeFileFS(w, r, spaDistRoot, "index.html")
		return
	}
	http.NotFound(w, r)
}

// spaOrDiskAssetsHandler serves from embedded FS first, then from disk (for local dev).
func spaOrDiskAssetsHandler(embedded fs.FS, diskDir string) http.Handler {
	disk := http.FileServer(http.Dir(diskDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path
		if name == "" || name == "/" {
			name = "index.html"
		} else if name[0] == '/' {
			name = name[1:]
		}
		if f, err := embedded.Open(name); err == nil {
			f.Close()
			http.FileServer(http.FS(embedded)).ServeHTTP(w, r)
			return
		}
		disk.ServeHTTP(w, r)
	})
}
