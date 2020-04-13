package main

import (
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"gopkg.in/mgo.v2"
)

var (
	//DB NovelDatabase
	OAuthConfig *oauth2.Config
	StorageBucket *storage.BucketHandle
	StorageBucketName string
	SessionStore sessions.Store
	PubsubClient *pubsub.Client
	_ mgo.Session
)

type cloudSQLConfig struct{
	Username string
	Password string
	Instance string
}

func configureClousSQL(config cloudSQLConfig)(NovelDatabase, error){

}