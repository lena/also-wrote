package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	BaseURL      = "https://api.themoviedb.org/3"
	ImageBaseURL = "https://image.tmdb.org/t/p/w500" // Use w500 for decent quality
	cacheTTL     = 10 * time.Minute                  // Cache TMDB responses to reduce API calls
)

type Client struct {
	Token      string
	HTTPClient *http.Client
	cache      *ttlCache
}

func NewClient(token string) *Client {
	return &Client{
		Token: token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: newTTLCache(cacheTTL),
	}
}

func (c *Client) doRequest(method, endpoint string, params url.Values, target interface{}) error {
	reqURL, err := url.Parse(BaseURL + endpoint)
	if err != nil {
		return err
	}

	if params != nil {
		reqURL.RawQuery = params.Encode()
	}

	req, err := http.NewRequest(method, reqURL.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// --- Structs ---

type SearchTVResponse struct {
	Page         int      `json:"page"`
	Results      []TVShow `json:"results"`
	TotalPages   int      `json:"total_pages"`
	TotalResults int      `json:"total_results"`
}

type TVShow struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	FirstAirDate string `json:"first_air_date"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
}

type TVShowDetails struct {
	TVShow
	NumberOfSeasons int      `json:"number_of_seasons"`
	Seasons         []Season `json:"seasons"`
	CreatedBy       []Person `json:"created_by"`
}

type Season struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Overview     string    `json:"overview"`
	SeasonNumber int       `json:"season_number"`
	PosterPath   string    `json:"poster_path"`
	Episodes     []Episode `json:"episodes,omitempty"`
}

type Episode struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Overview      string   `json:"overview"`
	AirDate       string   `json:"air_date"`
	EpisodeNumber int      `json:"episode_number"`
	SeasonNumber  int      `json:"season_number"`
	StillPath     string   `json:"still_path"`
	Crew          []Person `json:"crew"`
}

// Calling this Person instead of Writer because this struct is reused for other roles like Director
type Person struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Job                string `json:"job"`
	Character          string `json:"character"`
	Department         string `json:"department"`
	KnownForDepartment string `json:"known_for_department"`
	ProfilePath        string `json:"profile_path"`
}

type PersonCreditsResponse struct {
	Cast []Credit `json:"cast"`
	Crew []Credit `json:"crew"`
}

type Credit struct {
	ID           int    `json:"id"`
	CreditID     string `json:"credit_id"`
	Name         string `json:"name"`
	Character    string `json:"character"`
	Job          string `json:"job"`
	Department   string `json:"department"`
	EpisodeCount int    `json:"episode_count"`
	FirstAirDate string `json:"first_air_date"`
	PosterPath   string `json:"poster_path"`
	Overview     string `json:"overview"`
}

type CreditDetails struct {
	Media struct {
		Episodes []Episode `json:"episodes"`
	} `json:"media"`
}

type SearchPeopleResponse struct {
	Page         int      `json:"page"`
	Results      []Person `json:"results"`
	TotalPages   int      `json:"total_pages"`
	TotalResults int      `json:"total_results"`
}

type AggregateCreditsResponse struct {
	Cast []AggregateCredit `json:"cast"`
	Crew []AggregateCredit `json:"crew"`
}

type AggregateCredit struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Department  string `json:"department"`
	Jobs        []Job  `json:"jobs"`
	ProfilePath string `json:"profile_path"`
}

type Job struct {
	CreditID     string `json:"credit_id"`
	Job          string `json:"job"`
	EpisodeCount int    `json:"episode_count"`
}

// --- Methods ---

func (c *Client) SearchTVShows(query string) ([]TVShow, error) {
	key := "search_tv:" + strings.ToLower(strings.TrimSpace(query))
	if v, ok := c.cache.get(key); ok {
		return v.([]TVShow), nil
	}
	params := url.Values{}
	params.Add("query", query)
	params.Add("language", "en-US")

	var result SearchTVResponse
	err := c.doRequest("GET", "/search/tv", params, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, result.Results)
	return result.Results, nil
}

func (c *Client) SearchPeople(query string) ([]Person, error) {
	key := "search_people:" + strings.ToLower(strings.TrimSpace(query))
	if v, ok := c.cache.get(key); ok {
		return v.([]Person), nil
	}
	params := url.Values{}
	params.Add("query", query)
	params.Add("language", "en-US")

	var result SearchPeopleResponse
	err := c.doRequest("GET", "/search/person", params, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, result.Results)
	return result.Results, nil
}

func (c *Client) GetTVShowDetails(id int) (*TVShowDetails, error) {
	key := fmt.Sprintf("show:%d", id)
	if v, ok := c.cache.get(key); ok {
		return v.(*TVShowDetails), nil
	}
	var result TVShowDetails
	err := c.doRequest("GET", fmt.Sprintf("/tv/%d", id), nil, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, &result)
	return &result, nil
}

func (c *Client) GetSeasonDetails(tvID, seasonNumber int) (*Season, error) {
	key := fmt.Sprintf("season:%d:%d", tvID, seasonNumber)
	if v, ok := c.cache.get(key); ok {
		return v.(*Season), nil
	}
	var result Season
	endpoint := fmt.Sprintf("/tv/%d/season/%d", tvID, seasonNumber)
	err := c.doRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, &result)
	return &result, nil
}

func (c *Client) GetSeasonAggregateCredits(tvID, seasonNumber int) (*AggregateCreditsResponse, error) {
	key := fmt.Sprintf("season_credits:%d:%d", tvID, seasonNumber)
	if v, ok := c.cache.get(key); ok {
		return v.(*AggregateCreditsResponse), nil
	}
	var result AggregateCreditsResponse
	endpoint := fmt.Sprintf("/tv/%d/season/%d/aggregate_credits", tvID, seasonNumber)
	err := c.doRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, &result)
	return &result, nil
}

func (c *Client) GetPersonTVCredits(personID int) (*PersonCreditsResponse, error) {
	key := fmt.Sprintf("person_credits:%d", personID)
	if v, ok := c.cache.get(key); ok {
		return v.(*PersonCreditsResponse), nil
	}
	var result PersonCreditsResponse
	endpoint := fmt.Sprintf("/person/%d/tv_credits", personID)
	err := c.doRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, &result)
	return &result, nil
}

func (c *Client) GetPersonDetails(personID int) (*Person, error) {
	key := fmt.Sprintf("person:%d", personID)
	if v, ok := c.cache.get(key); ok {
		return v.(*Person), nil
	}
	var result Person
	endpoint := fmt.Sprintf("/person/%d", personID)
	err := c.doRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, &result)
	return &result, nil
}

func (c *Client) GetCreditDetails(creditID string) (*CreditDetails, error) {
	key := "credit:" + creditID
	if v, ok := c.cache.get(key); ok {
		return v.(*CreditDetails), nil
	}
	var result CreditDetails
	endpoint := fmt.Sprintf("/credit/%s", creditID)
	err := c.doRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, &result)
	return &result, nil
}

func (c *Client) GetEpisodeDetails(tvID, seasonNumber, episodeNumber int) (*Episode, error) {
	key := fmt.Sprintf("episode:%d:%d:%d", tvID, seasonNumber, episodeNumber)
	if v, ok := c.cache.get(key); ok {
		return v.(*Episode), nil
	}
	var result Episode
	endpoint := fmt.Sprintf("/tv/%d/season/%d/episode/%d", tvID, seasonNumber, episodeNumber)
	err := c.doRequest("GET", endpoint, nil, &result)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, &result)
	return &result, nil
}
