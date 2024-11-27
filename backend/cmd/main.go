package main

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"
)

type LoginCache struct {
	ID        string
	AP        string
	Timestamp time.Time
}

var (
	loginMap = make(map[string]LoginCache)
	mu       sync.Mutex
)

type LoginRequest struct {
	CacheID string `json:"cacheId"`
	Name    string `json:"username"`
	Email   string `json:"email"`
}

func main() {
	go purgeCacheEvery(30 * time.Second)
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found. Continuing without it.")
	}
	username := os.Getenv("UNIFI_USERNAME")
	password := os.Getenv("UNIFI_PASSWORD")
	url := os.Getenv("UNIFI_URL")
	site := os.Getenv("UNIFI_SITE")
	duration, err := strconv.Atoi(os.Getenv("UNIFI_DURATION"))
	if err != nil {
		log.Fatal("Error loading duration from env file")
	}
	disableTLS, err := strconv.ParseBool(os.Getenv("DISABLE_TLS"))
	if err != nil {
		disableTLS = false
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/api/login", func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest

		// Parse the JSON body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		// Extract cacheId
		cacheId := req.CacheID

		if cacheId != "" {
			cacheInfo := getRecord(cacheId)
			authorizeGuest(url, site, username, password, cacheInfo.ID, cacheInfo.AP, duration, disableTLS)
			writeToDb(cacheId, cacheInfo.ID, cacheInfo.AP, req.Name, req.Email, duration)
			removeFromCache(cacheId)
		}
		http.Redirect(w, r, "/success", http.StatusSeeOther)

	})
	r.Get("/success", func(w http.ResponseWriter, r *http.Request) {
		serveFrontend(w, r, "")
	})
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		var cacheId string
		if r.URL.Query().Get("id") != "" {
			id := r.URL.Query().Get("id")
			ap := r.URL.Query().Get("ap")
			cacheId = addToCache(id, ap)
		}
		serveFrontend(w, r, cacheId)
	})
	appUrl := fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))
	fmt.Printf("Serving application on %s", appUrl)
	http.ListenAndServe(appUrl, r)
}

func serveFrontend(w http.ResponseWriter, r *http.Request, cacheId string) {
	// Set the path to the build output of the front-end
	frontendDir := "./"
	debugMode, _ := strconv.ParseBool(os.Getenv("DEBUG_MODE"))
	if debugMode {
		frontendDir = "dist"
	}

	// Helper function to serve HTML pages
	serveHTML := func(fileName string, w http.ResponseWriter, r *http.Request, cacheId string) {
		// Read the HTML file
		filePath := filepath.Join(frontendDir, fileName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Inject cacheId into the HTML content if provided
		if cacheId != "" {
			fileContent = []byte(strings.Replace(string(fileContent), "</body>", fmt.Sprintf(`<script>window.cacheId = "%s";</script></body>`, cacheId), 1))
		}
		appName := os.Getenv("VITE_PAGE_TITLE")
		if appName == "" {
			fmt.Println("Error getting the page title. Falling back to default.")
			appName = "Guest Portal"
		}
		fileContent = []byte(strings.Replace(string(fileContent), "%VITE_PAGE_TITLE%", appName, -1))

		// Serve the HTML file with the injected cacheId
		w.Header().Set("Content-Type", "text/html")
		w.Write(fileContent)
	}

	// Handle serving the main entry point (index.html)
	if r.URL.Path == "/" || r.URL.Path == "" || r.URL.Path == "/guest/s/default/" {
		serveHTML("index.html", w, r, cacheId)
		return
	}

	// Handle the /success route
	if r.URL.Path == "/success" {
		serveHTML("success.html", w, r, cacheId)
		return
	}

	// Serve other static assets like CSS, JS, images, etc.
	filePath := filepath.Join(frontendDir, r.URL.Path)
	if _, err := os.Stat(filePath); err == nil {
		http.ServeFile(w, r, filePath)
	} else {
		http.NotFound(w, r)
	}
}

func writeToDb(cacheId string, id string, ap string, name string, email string, duration int) {
	// Open (or create) the SQLite database
	err := os.MkdirAll(os.Getenv("DB_PATH"), os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s/unifi-guest-portal.db", os.Getenv("DB_PATH")))
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure the table exists
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS user_sessions (
	    cache_id TEXT PRIMARY KEY,
		id TEXT,
		ap TEXT,
		name TEXT,
		email TEXT,
		duration INTEGER,
		created_at TEXT
	);`
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	currentTime := time.Now().Format(time.RFC3339)

	// Insert the data
	insertQuery := `INSERT INTO user_sessions (cache_id, id, ap, name, email, duration, created_at) 
					VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(insertQuery, cacheId, id, ap, name, email, duration, currentTime)
	if err != nil {
		log.Printf("Failed to insert data: %v", err)
	} else {
		log.Println("Data inserted successfully")
	}
}

func addToCache(id string, ap string) string {
	mu.Lock()
	defer mu.Unlock()

	cacheID := uuid.New().String()
	loginMap[cacheID] = LoginCache{ID: id, AP: ap, Timestamp: time.Now()}
	return cacheID
}

func removeFromCache(cacheID string) bool {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := loginMap[cacheID]; exists {
		delete(loginMap, cacheID)
		return true
	}
	return false
}

func getRecord(cacheID string) *LoginCache {
	mu.Lock()
	defer mu.Unlock()

	if entry, exists := loginMap[cacheID]; exists {
		return &entry
	}
	return nil
}

func purgeCacheEvery(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Use for range to handle the periodic ticker
	for range ticker.C {
		purgeCache()
	}
}

// Purge cache entries older than a threshold (e.g., 1 hour)
func purgeCache() {
	mu.Lock()
	defer mu.Unlock()

	threshold := time.Now().Add(-1 * time.Hour) // Purge entries older than 1 hour

	for cacheID, entry := range loginMap {
		if entry.Timestamp.Before(threshold) {
			delete(loginMap, cacheID)
			log.Printf("Purged cache entry: %s", cacheID)
		}
	}
}

func authorizeGuest(controllerURL, site, username, password, clientMAC string, apMAC string, duration int, disableTLS bool) error {
	// Step 1 Login to the router
	loginURL := fmt.Sprintf("%s/api/auth/login", controllerURL)
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	loginData, _ := json.Marshal(loginPayload)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: disableTLS,
			},
		},
	}
	req, err := http.NewRequest(http.MethodPost, loginURL, bytes.NewBuffer(loginData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to login to UniFi: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", string(body))
	}

	// Capture session cookie
	cookies := resp.Cookies()
	csrfToken := resp.Header.Get("x-csrf-token")

	// Step 2: Authorize the Guest
	authURL := fmt.Sprintf("%s/proxy/network/api/s/%s/cmd/stamgr", controllerURL, site)
	authPayload := map[string]interface{}{
		"cmd":     "authorize-guest",
		"mac":     clientMAC,
		"minutes": duration,
		"ap_mac":  apMAC,
	}
	authData, _ := json.Marshal(authPayload)

	req, err = http.NewRequest(http.MethodPost, authURL, bytes.NewBuffer(authData))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Attach the session cookie to the request
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	req.Header.Add("x-csrf-token", csrfToken)

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authorize guest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return fmt.Errorf("authorization failed: %s", string(body))
	}
	fmt.Println("Auth Sent")
	return nil
}
