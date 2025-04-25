package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
)

var (
	db    *sql.DB
	debug = os.Getenv("DEBUG") == "TRUE"
)

func main() {
	db = initializedDatabase()
	defer db.Close()
	startTokensCleaningService(db)

	http.HandleFunc("/tokens/get", RetrieveTokensHandler(db))
	http.HandleFunc("/tokens/refresh", RefreshTokensHandler(db))

	log.Println("Сервер запущен на порту :80")
	log.Fatal(http.ListenAndServe(":80", nil))
}
