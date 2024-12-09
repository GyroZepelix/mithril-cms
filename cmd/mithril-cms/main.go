package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/GyroZepelix/mithril-cms/internal/handlers"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/service/user"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/go-playground/validator/v10"
	_ "github.com/lib/pq"
)

func main() {
	logging.Init(os.Stdout)

	db := connectDB(
		config.Envs.DBDriver,
		config.Envs.DBUser,
		config.Envs.DBPassword,
		config.Envs.DBHost,
		config.Envs.DBPort,
		config.Envs.DBName,
		config.Envs.DBFlags,
	)
	defer db.Close()

	queries := persistence.New(db)
	env := &handlers.ServiceContext{
		UserManager: user.NewManager(queries),
		Validator:   validator.New(validator.WithRequiredStructEnabled()),
	}
	router := handlers.NewRouter(env)

	addr := fmt.Sprintf("%s:%s", config.Envs.PublicHost, config.Envs.Port)
	fmt.Printf("Server listening on %s\n", addr)

	err := http.ListenAndServe(addr, router)
	if err != nil {
		panic(err)
	}
}

func connectDB(dbdriver, dbuser, dbpassword, dbhost, dbport, dbname, dbflags string) *sql.DB {
	connectionString := fmt.Sprintf("%s://%s:%s@%s:%s/%s?%s",
		dbdriver,
		dbuser,
		dbpassword,
		dbhost,
		dbport,
		dbname,
		dbflags,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db
}
