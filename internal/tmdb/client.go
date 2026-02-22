package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL      = "https://api.themoviedb.org/3"
	ImageBaseURL = "https://image.tmdb.org/t/p/w500" // Use w500 for decent quality
)

type Client struct {
	Token      string
	HTTPClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		Token: token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
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
	params := url.Values{}
	params.Add("query", query)
	params.Add("language", "en-US")

	var result SearchTVResponse
	err := c.doRequest("GET", "/search/tv", params, &result)
	return result.Results, err
}

func (c *Client) SearchPeople(query string) ([]Person, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("language", "en-US")

	var result SearchPeopleResponse
	err := c.doRequest("GET", "/search/person", params, &result)
	return result.Results, err
}

func (c *Client) GetTVShowDetails(id int) (*TVShowDetails, error) {
	var result TVShowDetails
	err := c.doRequest("GET", fmt.Sprintf("/tv/%d", id), nil, &result)
	return &result, err
}

func (c *Client) GetSeasonDetails(tvID, seasonNumber int) (*Season, error) {
	var result Season
	endpoint := fmt.Sprintf("/tv/%d/season/%d", tvID, seasonNumber)
	err := c.doRequest("GET", endpoint, nil, &result)
	return &result, err
}

func (c *Client) GetSeasonAggregateCredits(tvID, seasonNumber int) (*AggregateCreditsResponse, error) {
	var result AggregateCreditsResponse
	endpoint := fmt.Sprintf("/tv/%d/season/%d/aggregate_credits", tvID, seasonNumber)
	err := c.doRequest("GET", endpoint, nil, &result)
	return &result, err
}

func (c *Client) GetPersonTVCredits(personID int) (*PersonCreditsResponse, error) {
	var result PersonCreditsResponse
	endpoint := fmt.Sprintf("/person/%d/tv_credits", personID)
	err := c.doRequest("GET", endpoint, nil, &result)
	return &result, err
}

func (c *Client) GetPersonDetails(personID int) (*Person, error) {
	var result Person
	endpoint := fmt.Sprintf("/person/%d", personID)
	err := c.doRequest("GET", endpoint, nil, &result)
	return &result, err
}

func (c *Client) GetCreditDetails(creditID string) (*CreditDetails, error) {
	var result CreditDetails
	endpoint := fmt.Sprintf("/credit/%s", creditID)
	err := c.doRequest("GET", endpoint, nil, &result)
	return &result, err
}

func (c *Client) GetEpisodeDetails(tvID, seasonNumber, episodeNumber int) (*Episode, error) {
	var result Episode
	endpoint := fmt.Sprintf("/tv/%d/season/%d/episode/%d", tvID, seasonNumber, episodeNumber)
	err := c.doRequest("GET", endpoint, nil, &result)
	return &result, err
}
