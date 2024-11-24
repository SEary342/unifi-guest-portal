package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

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
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
		if r.URL.Query().Get("id") != "" {
			authorizeGuest(url, site, username, password, r.URL.Query().Get("id"), r.URL.Query().Get("ap"), duration, disableTLS)
		}
	})
	http.ListenAndServe("0.0.0.0:80", r)
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
