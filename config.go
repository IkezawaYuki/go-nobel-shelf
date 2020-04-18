package main

import (
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"gopkg.in/mgo.v2"
	"log"
	"os"
)

var (
	DB                NovelDatabase
	OAuthConfig       *oauth2.Config
	StorageBucket     *storage.BucketHandle
	StorageBucketName string
	SessionStore      sessions.Store
	PubsubClient      *pubsub.Client
	_                 mgo.Session
)

const PubsubTopicID = "fill-novel-details"

type cloudSQLConfig struct {
	Username string
	Password string
	Instance string
}

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(err)
	}

	var err error

	// DB = newMemoryDB()
	DB, err = configureCloudSQL(cloudSQLConfig{
		Username: os.Getenv("USER"),
		Password: os.Getenv("PASSWORD"),
		Instance: os.Getenv("INSTANCE"),
	})

	if err != nil {
		log.Fatal(err)
	}

	cookieStore := sessions.NewCookieStore([]byte("something-very-secret"))
	cookieStore.Options = &sessions.Options{
		HttpOnly: true,
	}
	SessionStore = cookieStore

}

func configureCloudSQL(config cloudSQLConfig) (NovelDatabase, error) {
	if os.Getenv("GAE_INSTANCE") != "" {
		return newMySQLDB(MySQLConfig{
			Username:   config.Username,
			Password:   config.Password,
			UnixSocket: "/cloudsql/" + config.Instance,
		})
	}

	return newMySQLDB(MySQLConfig{
		Username: config.Username,
		Password: config.Password,
		Host:     "localhost",
		Port:     3306,
	})
}
