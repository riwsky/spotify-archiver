package main

import (
	spotify_archiver "github.com/riwsky/spotify-archiver"
	"log"
)

func main() {
	api := spotify_archiver.ApiFromEnv()
	loaded := api.DB
	for _, creds := range loaded.UserCreds {
		reauth, err := api.Auth(creds.RefreshToken, "refresh_token", "refresh_token")
		if err != nil {
			log.Fatalln(err)
		}
		reauth.RefreshToken = creds.RefreshToken
		log.Printf("Reauthed with %v", reauth)
		err = api.DB.Add(creds.Id, creds.Code, *reauth)
		if err != nil {
			log.Fatalln(err)
		}
		err = api.DB.Save()
		if err != nil {
			log.Fatalln(err)
		}
	}
	for _, creds := range api.DB.UserCreds {
		playlists := []string{"Discover Weekly"}
		if creds.Id == "riwsky" {
			playlists = []string{"Discover Weekly", "get to"}
		}
		log.Printf("Archiving %v for %v", playlists, creds.Id)
		spotify_archiver.Archive(creds, playlists)
	}
}
