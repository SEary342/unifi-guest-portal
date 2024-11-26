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
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"
)

type LoginCache struct {
	ID string
	AP string
}

var (
	loginMap = make(map[string]LoginCache)
	mu       sync.Mutex
)

type LoginRequest struct {
	CacheID string `json:"cacheId"`
	Name    string `json:"name"`
	Email   string `json:"email"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
			writeToDb(cacheInfo.ID, cacheInfo.AP, req.Name, req.Email, duration)
			removeFromCache(cacheId)
		}
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "" {
			id := r.URL.Query().Get("id")
			ap := r.URL.Query().Get("ap")
			cacheId := addToCache(id, ap)
			w.Write([]byte(cacheId))
		}
	})
	http.ListenAndServe("0.0.0.0:3000", r)
}

func writeToDb(id string, ap string, name string, email string, duration int) {
	// Open (or create) the SQLite database
	db, err := sql.Open("sqlite3", "app.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Ensure the table exists
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS user_sessions (
		id TEXT PRIMARY KEY,
		ap TEXT,
		name TEXT,
		email TEXT,
		duration INTEGER
	);`
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Insert the data
	insertQuery := `INSERT INTO user_sessions (id, ap, name, email, duration) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(insertQuery, id, ap, name, email, duration)
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
	loginMap[cacheID] = LoginCache{ID: id, AP: ap}
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
