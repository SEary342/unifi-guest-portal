// Package cache provides functions to manage a cache of login records.
// The cache stores login entries with a unique identifier and provides functionality to add, retrieve, remove, and periodically purge expired entries.
package cache

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LoginCache represents a cache entry for a login attempt.
// It contains the login details, the AP (Access Point) used, and the timestamp when the login was added.
type LoginCache struct {
	ID        string    // Unique identifier for the login entry.
	AP        string    // Access point (AP) associated with the login.
	Timestamp time.Time // Timestamp when the login entry was added.
}

var (
	// loginMap stores the cache entries with the cache ID as the key.
	loginMap = make(map[string]LoginCache)

	// mu is a mutex used to protect concurrent access to the loginMap.
	mu sync.Mutex
)

// AddToCache adds a new login entry to the cache and returns a unique cache ID.
//
// This function locks the cache during the operation to ensure thread-safety. The cache entry
// includes the provided ID, AP, and the current timestamp. A new cache ID is generated and returned.
func AddToCache(id string, ap string) string {
	mu.Lock()
	defer mu.Unlock()

	// Generate a new cache ID and store the login entry in the cache
	cacheID := uuid.New().String()
	loginMap[cacheID] = LoginCache{ID: id, AP: ap, Timestamp: time.Now()}
	return cacheID
}

// RemoveFromCache removes a login entry from the cache by its cache ID.
// It returns true if the entry was successfully removed, or false if the entry was not found.
//
// This function locks the cache during the operation to ensure thread-safety.
func RemoveFromCache(cacheID string) bool {
	mu.Lock()
	defer mu.Unlock()

	// Attempt to remove the cache entry, return true if successful, false otherwise
	if _, exists := loginMap[cacheID]; exists {
		delete(loginMap, cacheID)
		return true
	}
	return false
}

// GetRecord retrieves a login entry from the cache by its cache ID.
// It returns a pointer to the LoginCache entry if found, or nil if not found.
//
// This function locks the cache during the operation to ensure thread-safety.
func GetRecord(cacheID string) *LoginCache {
	mu.Lock()
	defer mu.Unlock()

	// Check if the cache entry exists, and return it if so
	if entry, exists := loginMap[cacheID]; exists {
		return &entry
	}
	return nil
}

// PurgeCacheEvery periodically purges cache entries older than a threshold.
// The interval specifies how frequently the cache should be purged (e.g., every 30 seconds).
//
// This function starts a ticker that runs at the specified interval and calls the `purgeCache`
// function periodically to clean up expired cache entries.
func PurgeCacheEvery(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Periodically check and purge old cache entries
	for range ticker.C {
		purgeCache()
	}
}

// purgeCache removes cache entries that are older than a specified threshold (1 hour in this case).
// This function is called periodically to keep the cache clean and prevent it from growing indefinitely.
//
// It locks the cache to safely iterate over the entries and removes those that are older than 1 hour.
func purgeCache() {
	mu.Lock()
	defer mu.Unlock()

	// Define the threshold: Purge entries older than 1 hour
	threshold := time.Now().Add(-1 * time.Hour)

	// Iterate over the cache entries and remove those that are older than the threshold
	for cacheID, entry := range loginMap {
		if entry.Timestamp.Before(threshold) {
			delete(loginMap, cacheID)
			log.Printf("Purged cache entry: %s", cacheID)
		}
	}
}
