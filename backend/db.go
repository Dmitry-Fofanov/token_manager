package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func initializedDatabase() (db *sql.DB) {

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@db/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal("Ошибка при открытии базы данных:", err)
	}

	if debug {
		guids := []string{
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
		}

		_, err = db.Exec(`
			INSERT INTO users (id, username, email)
			VALUES
				($1, 'user1', 'user1@example.com'),
				($2, 'user2', 'user2@example.com'),
				($3, 'user3', 'user3@example.com')
			ON CONFLICT (username) DO UPDATE
			SET id = excluded.id`,
			guids[0], guids[1], guids[2])

		log.Printf("Созданы пользователи с GUID:\n%s\n%s\n%s", guids[0], guids[1], guids[2])
	}

	return db
}

func startTokensCleaningService(db *sql.DB) {
	query := `
	DELETE FROM refresh_tokens
	WHERE expires_at < $1
	`
	ticker := time.NewTicker(time.Hour * 24)

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("Очищаю базу данных от просроченных токенов")
				_, err := db.Exec(query, time.Now())
				if err != nil {
					log.Printf("Не удалось очистить токены, ошибка: %w", err)
				}
			}
		}
	}()
}
