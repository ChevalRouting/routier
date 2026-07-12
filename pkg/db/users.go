package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

func ensureUsersSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL
	)`)
	return err
}

func seedDefaultUser(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	if count != 0 {
		return nil
	}

	initialPw := "admin"
	const seedFile = "/var/lib/routier/ui-seed-password"
	if data, err := os.ReadFile(seedFile); err == nil {
		if pw := strings.TrimSpace(string(data)); pw != "" {
			initialPw = pw
		}

		os.Remove(seedFile)
	}

	hash, err := HashPassword(initialPw)
	if err != nil {
		return fmt.Errorf("hash default password: %w", err)
	}

	if _, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", "routier", hash); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	_ = SetSetting(db, SettingPasswordChanged, "false")
	_ = SetSetting(db, SettingOnboardingComplete, "false")

	return nil
}

func AllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT id, username, password_hash FROM users")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash); err != nil {
			return nil, err
		}

		users = append(users, u)
	}

	return users, rows.Err()
}

func UpsertUser(db *sql.DB, username, passwordHash string) error {
	_, err := db.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)
		 ON CONFLICT(username) DO UPDATE SET password_hash = excluded.password_hash`,
		username, passwordHash,
	)
	return err
}

func UserByUsername(db *sql.DB, username string) (*User, error) {
	row := db.QueryRow("SELECT id, username, password_hash FROM users WHERE username = ?", username)
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &u, nil
}

func UpdatePassword(db *sql.DB, username, hash string) error {
	_, err := db.Exec("UPDATE users SET password_hash = ? WHERE username = ?", hash, username)
	return err
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
