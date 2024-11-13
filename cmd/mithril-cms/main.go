package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/GyroZepelix/mithril-cms/internal/routes"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgres://mithril:S3cret@localhost:5432/mithrildb?sslmode=disable"

	db := connectDB(connStr)
	defer db.Close()

	env := &routes.Env{
		DB: persistence.New(db),
	}
	router := routes.NewRouter(env)

	port := 8080
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server listening on http://localhost%s\n", addr)

	err := http.ListenAndServe(addr, router)
	if err != nil {
		panic(err)
	}
}

func connectDB(connectionString string) *sql.DB {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db
}
