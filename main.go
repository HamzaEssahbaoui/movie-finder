package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Constants for API endpoints
const (
	baseURL        = "https://api.themoviedb.org/3"
	searchEndpoint = "/search/movie"
	movieEndpoint  = "/movie/"
)

// Config struct to hold application configuration.
// It's good practice to keep configuration separate from your code logic.
type Config struct {
	APIKey string
}

// Movie represents the basic information about a movie to be listed.
type Movie struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Year  string `json:"release_date"`
}

// MovieDetail represents the detailed information about a movie for display.
type MovieDetail struct {
	Title    string `json:"title"`
	Overview string `json:"overview"`
	// Add more fields as needed for detailed information.
}

// SearchResults wraps the list of movies returned by the API.
type SearchResults struct {
	Results []Movie `json:"results"`
}

// Initialize a template
var tmpl = template.Must(template.New("movie").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Overview}}</p>
</body>
</html>
`))

func main() {
	// Securely manage the API key using environment variables.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		log.Fatal("API key not set in TMDB_API_KEY environment variable")
	}
	config := Config{APIKey: apiKey}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(w, r, config)
	})
	http.HandleFunc("/movie/", func(w http.ResponseWriter, r *http.Request) {
		movieDetailsHandler(w, r, config) // Note the trailing slash for correct routing.
	})

	log.Println("Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request, config Config) {
	// Set the Content-Type header to ensure correct rendering of HTML.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Movie Finder</title>
		</head>
		<body>
			<h1>Search Movie Title</h1>
			<form action="/" method="GET">
				<input type="text" name="keyword" required>
				<button type="submit">Search</button>
			</form>
	`)

	// Extract the keyword from the query parameters.
	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		movies, err := searchMovies(keyword, config.APIKey)
		if err != nil {
			log.Printf("Error searching movies: %v", err)
			http.Error(w, "Failed to search movies", http.StatusInternalServerError)
			return
		}

		// Iterate through the search results and create links for detailed view.
		for _, movie := range movies.Results {
			fmt.Fprintf(w, "<p><a href=\"/movie/%d\">%s (%s)</a></p>", movie.ID, movie.Title, movie.Year)
		}
	}

	fmt.Fprintf(w, "</body></html>")
}

func movieDetailsHandler(w http.ResponseWriter, r *http.Request, config Config) {
	// Extracting the movie ID from the URL path.
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}
	movieID := pathParts[2]

	// Fetching movie details using the extracted ID.
	movie, err := fetchMovieDetails(movieID, config.APIKey)
	if err != nil {
		log.Printf("Error fetching movie details: %v", err)
		http.Error(w, "Failed to fetch movie details", http.StatusInternalServerError)
		return
	}

	// Render the movie details using the template.
	if err := tmpl.Execute(w, movie); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func searchMovies(keyword string, apiKey string) (*SearchResults, error) {
	requestURL := fmt.Sprintf("%s%s?api_key=%s&query=%s", baseURL, searchEndpoint, apiKey, url.QueryEscape(keyword))
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results SearchResults
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	return &results, nil
}

func fetchMovieDetails(movieID string, apiKey string) (*MovieDetail, error) {
	requestURL := fmt.Sprintf("%s%s%s?api_key=%s", baseURL, movieEndpoint, movieID, apiKey)
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var movieDetail MovieDetail
	if err := json.NewDecoder(resp.Body).Decode(&movieDetail); err != nil {
		return nil, err
	}

	return &movieDetail, nil
}
