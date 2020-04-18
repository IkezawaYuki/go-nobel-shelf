package infrastructure

import (
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"fmt"
	"github.com/IkezawaYuki/go-novel-shelf/datastore"
	"github.com/IkezawaYuki/go-novel-shelf/domain"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"gopkg.in/mgo.v2"
	"log"
	"os"
)

var (
	DB                domain.NovelDatabase
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
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(err)
	}

	var err error

	//DB = datastore.NewMemoryDB()
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

func configureCloudSQL(config cloudSQLConfig) (domain.NovelDatabase, error) {
	if os.Getenv("GAE_INSTANCE") != "" {
		return datastore.NewMySQLDB(datastore.MySQLConfig{
			Username:   config.Username,
			Password:   config.Password,
			UnixSocket: "/cloudsql/" + config.Instance,
		})
	}
	fmt.Println("configureCloudSQL")
	return datastore.NewMySQLDB(datastore.MySQLConfig{
		Username: config.Username,
		Password: config.Password,
		Host:     "localhost",
		Port:     3306,
	})
}
