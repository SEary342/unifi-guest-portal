package authorization

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AuthorizeGuestProcess orchestrates the process of logging into the UniFi
// controller and authorizing a guest.
//
// It first calls the login function to obtain the session cookies and CSRF token,
// and then uses that information to call the authorizeGuest function.
//
// Parameters:
//   - controllerURL: The base URL of the UniFi controller.
//   - site: The site to which the guest should be authorized.
//   - username: The username used for logging in.
//   - password: The password used for logging in.
//   - clientMAC: The MAC address of the client to be authorized.
//   - apMAC: The MAC address of the access point to which the client is connected.
//   - duration: The duration (in minutes) for which the guest will be authorized.
//   - disableTLS: A flag indicating whether to skip TLS verification (for insecure connections).
//
// Returns:
//   - error: An error if any of the steps fail, otherwise nil.
func AuthorizeGuestProcess(controllerURL, site, username, password, clientMAC, apMAC string, duration int, disableTLS bool) error {
	// Login to the router and retrieve session cookies and CSRF token
	cookies, csrfToken, err := login(controllerURL, username, password, disableTLS)
	if err != nil {
		return err
	}

	// Authorize the guest using the session cookies and CSRF token
	err = authorizeGuest(controllerURL, site, clientMAC, apMAC, duration, cookies, csrfToken)
	if err != nil {
		return err
	}

	return nil
}

// login handles the login process to the UniFi controller.
//
// It sends a POST request with the provided username and password to the
// controller's login endpoint and retrieves the session cookies and CSRF token.
//
// Parameters:
//   - controllerURL: The base URL of the UniFi controller.c
//   - username: The username used for logging in.
//   - password: The password used for logging in.
//   - disableTLS: A flag indicating whether to skip TLS verification (for insecure connections).
//
// Returns:
//   - []*http.Cookie: The session cookies returned by the login request.
//   - string: The CSRF token needed for subsequent requests.
//   - error: An error if the login fails, otherwise nil.
func login(controllerURL, username, password string, disableTLS bool) ([]*http.Cookie, string, error) {
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
		return nil, "", fmt.Errorf("failed to create login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to login to UniFi: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("login failed: %s", string(body))
	}

	cookies := resp.Cookies()
	csrfToken := resp.Header.Get("x-csrf-token")

	return cookies, csrfToken, nil
}

// authorizeGuest sends a request to authorize a guest on the UniFi controller.
//
// It sends a POST request to the controller's `stamgr` endpoint with the provided
// guest information and session details (cookies and CSRF token).
//
// Parameters:
//   - controllerURL: The base URL of the UniFi controller.
//   - site: The site to which the guest should be authorized.
//   - clientMAC: The MAC address of the client to be authorized.
//   - apMAC: The MAC address of the access point to which the client is connected.
//   - duration: The duration (in minutes) for which the guest will be authorized.
//   - cookies: The session cookies obtained from a successful login request.
//   - csrfToken: The CSRF token required for authorization.
//
// Returns:
//   - error: An error if the authorization fails, otherwise nil.
func authorizeGuest(controllerURL, site, clientMAC, apMAC string, duration int, cookies []*http.Cookie, csrfToken string) error {
	authURL := fmt.Sprintf("%s/proxy/network/api/s/%s/cmd/stamgr", controllerURL, site)
	authPayload := map[string]interface{}{
		"cmd":     "authorize-guest",
		"mac":     clientMAC,
		"minutes": duration,
		"ap_mac":  apMAC,
	}
	authData, _ := json.Marshal(authPayload)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, authURL, bytes.NewBuffer(authData))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Attach session cookies and CSRF token to the request
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	req.Header.Add("x-csrf-token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authorize guest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authorization failed: %s", string(body))
	}
	fmt.Println("Auth Sent")
	return nil
}
