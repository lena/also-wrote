// One-off: list each user and their favorite-writer count.
// Usage (from repo root):
//   DATABASE_URL='postgresql://user:pass@host/db?sslmode=require' go run ./cmd/list-users
// Or put DATABASE_URL in .env and run: go run ./cmd/list-users
package main

import (
	"also-wrote/internal/db"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	loadEnv()
	conn := os.Getenv("DATABASE_URL")
	if conn == "" {
		fmt.Fprintln(os.Stderr, "Set DATABASE_URL (e.g. from Render External URL) or add it to .env")
		os.Exit(1)
	}
	database, err := db.Open(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	rows, err := database.Query(`
		SELECT u.id, u.email, COUNT(f.person_id) AS n
		FROM users u
		LEFT JOIN user_fave_writers f ON f.user_id = u.id
		GROUP BY u.id, u.email
		ORDER BY u.id
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Println("id | email | favorite_writers_count")
	fmt.Println("---|-------|------------------------")
	for rows.Next() {
		var id int64
		var email string
		var n int
		if err := rows.Scan(&id, &email, &n); err != nil {
			fmt.Fprintf(os.Stderr, "Scan: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%d | %s | %d\n", id, email, n)
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Rows: %v\n", err)
		os.Exit(1)
	}
}

func loadEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
