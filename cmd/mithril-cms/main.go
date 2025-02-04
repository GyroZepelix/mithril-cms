package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GyroZepelix/mithril-cms/internal/config"
	"github.com/GyroZepelix/mithril-cms/internal/constant"
	"github.com/GyroZepelix/mithril-cms/internal/handlers"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/middleware"
	"github.com/GyroZepelix/mithril-cms/internal/response"
	"github.com/GyroZepelix/mithril-cms/internal/service/content"
	"github.com/GyroZepelix/mithril-cms/internal/service/permission"
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

	env := setupEnv(db)
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

func setupEnv(db *sql.DB) *handlers.ServiceContext {
	queries := persistence.New(db)

	userManager := user.NewManager(queries)
	contentManager := content.NewManager(queries)
	ownershipChecker := permission.NewOwnershipChecker(contentManager)
	unauthorizedResponse := func(w http.ResponseWriter) { response.Unauthorized(w, "Insufficient Permissions") }

	permissionValidator := permission.NewPermissionValidator()
	setupPermissions(permissionValidator)

	return &handlers.ServiceContext{
		UserManager:          userManager,
		ContentManager:       contentManager,
		PermissionMiddleware: middleware.NewPermissionMiddleware("id", unauthorizedResponse, ownershipChecker, permissionValidator),
		Validator:            validator.New(validator.WithRequiredStructEnabled()),
	}
}

func setupPermissions(pm permission.PermissionValidator) {

	readerPermissions := []permission.AccessPermission{
		{
			ResourceType:    permission.ResourceTypeUser,
			Permission:      permission.CanRead,
			PermissionLevel: permission.Owned,
		},
	}
	authorPermissions := append(readerPermissions,
		[]permission.AccessPermission{
			{
				ResourceType:    permission.ResourceTypePost,
				Permission:      permission.CanRead,
				PermissionLevel: permission.Owned,
			},
			{
				ResourceType:    permission.ResourceTypePost,
				Permission:      permission.CanCreate,
				PermissionLevel: permission.Owned,
			},
			{
				ResourceType:    permission.ResourceTypePost,
				Permission:      permission.CanDelete,
				PermissionLevel: permission.Owned,
			},
			{
				ResourceType:    permission.ResourceTypePost,
				Permission:      permission.CanUpdate,
				PermissionLevel: permission.Owned,
			},
		}...,
	)

	adminPermissions := append(authorPermissions,
		[]permission.AccessPermission{
			// Users
			{
				ResourceType:    permission.ResourceTypeUser,
				Permission:      permission.CanRead,
				PermissionLevel: permission.All,
			},
			{
				ResourceType:    permission.ResourceTypeUser,
				Permission:      permission.CanDelete,
				PermissionLevel: permission.All,
			},
			{
				ResourceType:    permission.ResourceTypeUser,
				Permission:      permission.CanUpdate,
				PermissionLevel: permission.All,
			},
			// Posts
			{
				ResourceType:    permission.ResourceTypePost,
				Permission:      permission.CanRead,
				PermissionLevel: permission.All,
			},
			{
				ResourceType:    permission.ResourceTypePost,
				Permission:      permission.CanDelete,
				PermissionLevel: permission.All,
			},
			{
				ResourceType:    permission.ResourceTypePost,
				Permission:      permission.CanUpdate,
				PermissionLevel: permission.All,
			},
		}...,
	)

	pm.RegisterRole(constant.UserRoleReader, readerPermissions...)
	pm.RegisterRole(constant.UserRoleAuthor, authorPermissions...)
	pm.RegisterRole(constant.UserRoleAdmin, adminPermissions...)
}
