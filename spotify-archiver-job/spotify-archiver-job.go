package main

import (
	spotify_archiver "github.com/riwsky/spotify-archiver"
	"github.com/riwsky/spotify-archiver/db"
	"log"
)

func main() {
	loaded, err := db.Load()
	if err != nil {
		log.Fatal(err)
	}
	for _, creds := range loaded.UserCreds {
		log.Printf("Archiving %v", creds.Id)
		playlists := []string{"Discover Weekly"}
		if creds.Id == "riwsky" {
			playlists = []string{"Discover Weekly", "get to"}
		}
		spotify_archiver.Archive(creds, playlists)
	}
}
