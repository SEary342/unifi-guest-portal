// Package db provides functionality for database interactions, specifically
// for managing user session records in a SQLite database.
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

// WriteToDb inserts a user session record into the SQLite database. If the database or its
// table does not exist, they will be created automatically.
//
// Parameters:
// - cacheId: Unique identifier for the cached session.
// - id: User or device identifier associated with the session.
// - ap: Access point identifier for the session.
// - name: Name of the user or device owner.
// - email: Email address of the user.
// - duration: Session duration in minutes.
//
// Environment Variables:
//   - DB_PATH: The file path where the SQLite database is stored. If the directory does not exist,
//     it will be created.
//
// Behavior:
// - Creates a directory for the database if it does not already exist.
// - Opens (or creates) a SQLite database named `unifi-guest-portal.db` in the specified `DB_PATH`.
// - Ensures a `user_sessions` table exists with the following schema:
//   - cache_id (TEXT PRIMARY KEY): Unique session identifier.
//   - id (TEXT): User or device identifier.
//   - ap (TEXT): Access point identifier.
//   - name (TEXT): User or device owner's name.
//   - email (TEXT): User's email address.
//   - duration (INTEGER): Session duration in minutes.
//   - created_at (TEXT): Timestamp when the record was created in RFC3339 format.
//
// - Inserts a new record into the `user_sessions` table with the provided parameters.
//
// Errors:
// - Logs and terminates the application if the database cannot be opened or the table cannot be created.
// - Logs a warning if data insertion fails.
//
// Example:
// ```go
// err := os.Setenv("DB_PATH", "/path/to/db")
//
//	if err != nil {
//	    log.Fatalf("Failed to set DB_PATH: %v", err)
//	}
//
// db.WriteToDb("cache123", "id456", "ap789", "John Doe", "john@example.com", 120)
// ```
func WriteToDb(cacheId string, id string, ap string, name string, email string, duration int) {
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
