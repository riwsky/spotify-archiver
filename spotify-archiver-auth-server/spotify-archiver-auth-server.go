package main

import (
	spotify_archiver "github.com/riwsky/spotify-archiver"
	"github.com/riwsky/spotify-archiver/db"
	"log"
	"net"
	"net/http"
	"os"
)

func mustHaveEnv(varName string) string {
	out := os.Getenv(varName)
	if out == "" {
		log.Fatal("couldn't find ", varName, " in environment")
	}
	return out
}

func main() {
	log.Println("auth server started")
	loaded, err := db.Load()
	if err != nil {
		log.Fatal(err)
	}
	api := spotify_archiver.Api{
		ClientId:     mustHaveEnv("SPOTIFY_ARCHIVER_CLIENT_ID"),
		ClientSecret: mustHaveEnv("SPOTIFY_ARCHIVER_CLIENT_SECRET"),
		DB:           loaded,
	}

	http.HandleFunc("/callback", api.Callback)
	http.Handle("/spotify_archiver", &api)
	ln, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.Serve(ln, http.DefaultServeMux))
}
