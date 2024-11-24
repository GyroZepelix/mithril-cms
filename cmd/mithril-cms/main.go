package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GyroZepelix/mithril-cms/internal/handlers"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/service/user"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/go-playground/validator/v10"
	_ "github.com/lib/pq"
)

func main() {
	logging.Init(os.Stdout)

	connStr := "postgres://mithril:S3cret@localhost:5432/mithrildb?sslmode=disable"
	db := connectDB(connStr)
	defer db.Close()

	queries := persistence.New(db)
	env := &handlers.Env{
		UserManager: user.NewManager(queries),
		Validator:   validator.New(validator.WithRequiredStructEnabled()),
	}
	router := handlers.NewRouter(env)

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
