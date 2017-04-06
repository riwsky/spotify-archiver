package main

import (
	spotify_archiver "github.com/riwsky/spotify-archiver"
	"log"
	"net"
	"net/http"
)

func main() {
	log.Println("auth server started")
	api := spotify_archiver.ApiFromEnv()

	http.HandleFunc("/callback", api.Callback)
	http.Handle("/spotify_archiver", &api)
	ln, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.Serve(ln, http.DefaultServeMux))
}
