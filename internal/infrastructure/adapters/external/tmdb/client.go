package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/narwhalmedia/narwhal/internal/domain/media"
)

// Client represents a TMDB API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new TMDB client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// MovieDetails represents the TMDB movie details response
type MovieDetails struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Overview    string   `json:"overview"`
	ReleaseDate string   `json:"release_date"`
	Genres      []Genre  `json:"genres"`
	Runtime     int      `json:"runtime"`
	PosterPath  string   `json:"poster_path"`
	Credits     Credits  `json:"credits"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

type CastMember struct {
	Name string `json:"name"`
	Role string `json:"character"`
}

type CrewMember struct {
	Name     string `json:"name"`
	Job      string `json:"job"`
	Director bool   `json:"-"`
}

// GetMovieDetails retrieves movie details from TMDB
func (c *Client) GetMovieDetails(ctx context.Context, tmdbID int) (*MovieDetails, error) {
	url := fmt.Sprintf("%s/movie/%d?api_key=%s&append_to_response=credits", c.baseURL, tmdbID, c.apiKey)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var details MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Process crew to identify director
	for i := range details.Credits.Crew {
		if details.Credits.Crew[i].Job == "Director" {
			details.Credits.Crew[i].Director = true
		}
	}

	return &details, nil
}

// ToDomainMovie converts TMDB movie details to our domain model
func (d *MovieDetails) ToDomainMovie() (*media.Movie, error) {
	// Parse release date
	releaseDate, err := time.Parse("2006-01-02", d.ReleaseDate)
	if err != nil {
		return nil, fmt.Errorf("parsing release date: %w", err)
	}

	// Extract genres
	genres := make([]string, len(d.Genres))
	for i, g := range d.Genres {
		genres[i] = g.Name
	}

	// Extract cast
	cast := make([]string, 0, len(d.Credits.Cast))
	for _, c := range d.Credits.Cast {
		cast = append(cast, c.Name)
	}

	// Find director
	var director string
	for _, c := range d.Credits.Crew {
		if c.Director {
			director = c.Name
			break
		}
	}

	// Create movie metadata
	metadata := media.NewMetadata(
		genres,
		director,
		cast,
		0, // Rating not available from TMDB
		"en", // Default language
	)

	// Create movie
	movie := media.NewMovie(
		d.Title,
		d.Overview,
		releaseDate,
		genres,
		director,
		cast,
	)

	// Set duration in seconds
	duration, err := media.NewDuration(d.Runtime * 60)
	if err != nil {
		return nil, fmt.Errorf("creating duration: %w", err)
	}
	movie.UpdateDuration(duration.Seconds())

	return movie, nil
} 