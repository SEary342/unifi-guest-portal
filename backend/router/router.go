// Package router defines the HTTP server setup, route handling, and the front-end serving logic
// for the Unifi Guest Portal application.
package router

import (
	"backend/authorization"
	"backend/cache"
	"backend/config"
	"backend/db"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/mattn/go-sqlite3"
)

// LoginRequest represents the structure of the JSON body for the login API.
type LoginRequest struct {
	CacheID string `json:"cacheId"`  // Cache identifier
	Name    string `json:"username"` // User's name
	Email   string `json:"email"`    // User's email address
}

// SetupServer initializes the HTTP server and defines application routes.
//
// Parameters:
// - cfg: Configuration object containing environment-specific settings.
//
// Routes:
// - POST /api/login: Handles guest login requests.
// - GET /success: Serves the success page.
// - GET /*: Serves the front-end assets or dynamically injects content.
//
// The server listens on the port specified in the configuration.
func SetupServer(cfg config.Config) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/api/login", func(w http.ResponseWriter, r *http.Request) {
		handleGuestAuthorization(w, r, cfg.URL, cfg.Site, cfg.Username, cfg.Password, cfg.Duration, cfg.DisableTLS)
	})

	r.Get("/success", func(w http.ResponseWriter, r *http.Request) {
		serveFrontend(w, r, "")
	})

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		var cacheId string
		if r.URL.Query().Get("id") != "" {
			id := r.URL.Query().Get("id")
			ap := r.URL.Query().Get("ap")
			cacheId = cache.AddToCache(id, ap)
		}
		serveFrontend(w, r, cacheId)
	})

	appUrl := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	fmt.Printf("Serving application on %s\n", appUrl)
	http.ListenAndServe(appUrl, r)
}

// serveFrontend serves the front-end assets and injects dynamic content as needed.
//
// Parameters:
// - w: HTTP response writer.
// - r: HTTP request.
// - cacheId: Cache identifier to inject into the front-end, if applicable.
//
// Behavior:
// - Serves `index.html` for the root route or default guest routes.
// - Serves `success.html` for the `/success` route.
// - Serves static assets like CSS, JS, or images for other routes.
// - Dynamically replaces placeholders in HTML files with runtime values (e.g., `cacheId` and app name).
func serveFrontend(w http.ResponseWriter, r *http.Request, cacheId string) {
	frontendDir := "./"
	debugMode, _ := strconv.ParseBool(os.Getenv("DEBUG_MODE"))
	if debugMode {
		frontendDir = "dist"
	}

	serveHTML := func(fileName string, w http.ResponseWriter, r *http.Request, cacheId string) {
		filePath := filepath.Join(frontendDir, fileName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if cacheId != "" {
			fileContent = []byte(strings.Replace(string(fileContent), "</body>",
				fmt.Sprintf(`<script>window.cacheId = "%s";</script></body>`, cacheId), 1))
		}
		appName := os.Getenv("VITE_PAGE_TITLE")
		if appName == "" {
			fmt.Println("Error getting the page title. Falling back to default.")
			appName = "Unifi Guest Portal"
		}
		fileContent = []byte(strings.Replace(string(fileContent), "%VITE_PAGE_TITLE%", appName, -1))

		w.Header().Set("Content-Type", "text/html")
		w.Write(fileContent)
	}

	if r.URL.Path == "/" || r.URL.Path == "" || r.URL.Path == "/guest/s/default/" {
		serveHTML("index.html", w, r, cacheId)
		return
	}

	if r.URL.Path == "/success" {
		serveHTML("success.html", w, r, cacheId)
		return
	}

	filePath := filepath.Join(frontendDir, r.URL.Path)
	if _, err := os.Stat(filePath); err == nil {
		http.ServeFile(w, r, filePath)
	} else {
		http.NotFound(w, r)
	}
}

// handleGuestAuthorization handles the POST /api/login requests to authorize a guest.
//
// Parameters:
// - w: HTTP response writer.
// - r: HTTP request.
// - url, site, username, password: Credentials and URL for Unifi API.
// - duration: Session duration.
// - disableTLS: Whether to disable TLS verification.
//
// Behavior:
// - Decodes the JSON body of the request.
// - Retrieves cache details and processes guest authorization.
// - Writes the session to the database and removes it from the cache.
// - Redirects the client to the `/success` page.
func handleGuestAuthorization(w http.ResponseWriter, r *http.Request, url, site, username, password string, duration int, disableTLS bool) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	cacheId := req.CacheID
	if cacheId != "" {
		cacheInfo := cache.GetRecord(cacheId)
		err := authorization.AuthorizeGuestProcess(url, site, username, password, cacheInfo.ID, cacheInfo.AP, duration, disableTLS)
		if err != nil {
			fmt.Println(err)
		}
		db.WriteToDb(cacheId, cacheInfo.ID, cacheInfo.AP, req.Name, req.Email, duration)
		cache.RemoveFromCache(cacheId)
	}

	http.Redirect(w, r, "/success", http.StatusSeeOther)
}
