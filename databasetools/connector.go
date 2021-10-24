package databasetools

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectToDatabase() *sqlx.DB {
	postgresUsername := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	databaseName := os.Getenv("POSTGRES_DB")
	connAsUrl := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", postgresUsername, postgresPassword, postgresHost, databaseName)
	db, err := sqlx.Open("postgres", connAsUrl)
	if err != nil {
		panic(err.Error())
	}
	return db
}
