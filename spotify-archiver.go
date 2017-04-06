package spotify_archiver

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/riwsky/spotify-archiver/db"
	"github.com/riwsky/spotify-archiver/types"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Api struct {
	ClientId     string
	ClientSecret string
	DB           *db.DB
	Mutex        sync.Mutex
}

func mustHaveEnv(varName string) string {
	out := os.Getenv(varName)
	if out == "" {
		log.Fatal("couldn't find ", varName, " in environment")
	}
	return out
}

func ApiFromEnv() Api {
	loaded, err := db.Load()
	if err != nil {
		log.Fatal(err)
	}
	return Api{
		ClientId:     mustHaveEnv("SPOTIFY_ARCHIVER_CLIENT_ID"),
		ClientSecret: mustHaveEnv("SPOTIFY_ARCHIVER_CLIENT_SECRET"),
		DB:           loaded,
	}
}

func tracks(creds db.PerUserCreds, playlist types.PlaylistResponseItem, offset, limit int) (*types.TrackResponse, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/users/%v/playlists/%v/tracks", playlist.Owner.Id, playlist.Id)
	var trackResponse types.TrackResponse
	err := get(creds, url, offset, limit, &trackResponse)
	if err != nil {
		return nil, err
	}
	return &trackResponse, nil
}

func addAuth(creds db.PerUserCreds, req *http.Request) {
	if req.Header == nil {
		req.Header = make(map[string][]string, 0)
	}
	req.Header["Authorization"] = []string{"Bearer " + creds.AccessToken}
}

func get(creds db.PerUserCreds, url string, offset, limit int, v interface{}) error {
	log.Println("getting to ", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	vals := req.URL.Query()
	vals.Add("offset", strconv.Itoa(offset))
	vals.Add("limit", strconv.Itoa(limit))
	req.URL.RawQuery = vals.Encode()
	req.Header = map[string][]string{
		"Accept": {"application/json"},
	}
	addAuth(creds, req)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func playlists(creds db.PerUserCreds, offset, limit int) (*types.PlaylistResponse, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/users/%v/playlists", creds.Id)
	var pitems types.PlaylistResponse
	err := get(creds, url, offset, limit, &pitems)
	if err != nil {
		return nil, fmt.Errorf("playlists: %v", err)
	}
	return &pitems, nil
}

const redirectUri = "http://riwsky.duckdns.org/callback"

func (s *Api) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Println("got hit.")
	u, err := url.Parse("https://accounts.spotify.com/authorize")
	if err != nil {
		log.Fatalln(err)
	}
	scope := "playlist-read-private playlist-read-collaborative playlist-modify-public playlist-modify-private"
	u.RawQuery = url.Values{
		"response_type": {"code"},
		"client_id":     {s.ClientId},
		"redirect_uri":  {redirectUri},
		"scope":         {scope},
	}.Encode()

	http.Redirect(w, req, u.String(), http.StatusSeeOther)
}

func do(req *http.Request, unmarshal func([]byte) error, tag string) error {
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting %v: %v", tag, err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
	if err != nil {
		return fmt.Errorf("error reading %v: %v", tag, err)
	}
	err = unmarshal(body)
	if err != nil {
		return fmt.Errorf("error unmarshaling %v: %v", tag, err)
	}
	return nil
}

func (s *Api) Auth(code, grantType, grantKey string) (*types.AuthResponse, error) {
	log.Println("ClientId/secret " + s.ClientId + " " + s.ClientSecret)
	urlStr := "https://accounts.spotify.com/api/token"
	log.Println("Attempting to auth with ", code)
	form := url.Values{
		grantKey:       {code},
		"redirect_uri": {redirectUri},
		"grant_type":   {grantType},
	}
	tokenReq, err := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(form.Encode()))
	b64ed := base64.StdEncoding.EncodeToString([]byte(s.ClientId + ":" + s.ClientSecret))
	tokenReq.Header = map[string][]string{
		"Authorization": {"Basic " + b64ed},
		"Content-Type":  {"application/x-www-form-urlencoded"},
	}
	var auth types.AuthResponse
	err = do(tokenReq, func(body []byte) error { return json.Unmarshal(body, &auth) }, "access and refresh token")
	if err != nil {
		return nil, err
	}
	return &auth, nil
}

func getId(creds db.PerUserCreds) (string, error) {
	url := "https://api.spotify.com/v1/me"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	addAuth(creds, req)
	var profile struct {
		Id string
	}
	err = do(req, func(body []byte) error { return json.Unmarshal(body, &profile) }, "user id")
	if err != nil {
		return "", err
	}
	return profile.Id, nil
}

func (s *Api) Callback(w http.ResponseWriter, req *http.Request) {
	log.Println("callback hit")
	err := req.ParseForm()
	if err != nil {
		log.Fatalln(err)
	}
	code := req.Form["code"][0]
	log.Println("got code " + code)
	auth, err := s.Auth(code, "authorization_code", "code")
	if err != nil {
		log.Fatalln(err)
	}
	id, err := getId(db.PerUserCreds{AccessToken: auth.AccessToken, RefreshToken: auth.RefreshToken})
	if err != nil {
		log.Fatalln(err)
	}
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	err = s.DB.Add(id, code, *auth)
	if err != nil {
		log.Fatalln(err)
	}
	s.DB.Save()
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "Spotify auth complete! Should now start archiving discover weekly for "+id)
}

func getAll(do func(offset, limit int) (*types.Paging, error)) error {
	currentOffset := 0
	limit := 50
	currentTotal := 100
	for currentOffset < currentTotal {
		paging, err := do(currentOffset, limit)
		if err != nil {
			return err
		}
		currentTotal = paging.Total
		currentOffset = currentOffset + limit
	}
	return nil
}

func getAllTracks(creds db.PerUserCreds, playlist types.PlaylistResponseItem) ([]string, error) {
	out := make([]string, 0)
	err := getAll(func(offset, limit int) (*types.Paging, error) {
		resp, err := tracks(creds, playlist, offset, limit)
		if err != nil {
			return nil, err
		}
		for _, track := range resp.Items {
			out = append(out, track.Track.URI)
		}
		return &resp.Paging, nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func Archive(creds db.PerUserCreds, playlists []string) error {
	plists, err := getAllPlaylists(creds)
	if err != nil {
		return fmt.Errorf("archiving: %v", err)
	}
	if len(plists) == 0 {
		return fmt.Errorf("User %v had no playlists", creds)
	}
	pListsByName := make(map[string]types.PlaylistResponseItem, len(plists))
	for _, plist := range plists {
		pListsByName[plist.Name] = plist
	}
	now := time.Now()
	toClones := make([]types.PlaylistResponseItem, 0)
	for _, playlist := range playlists {
		if shouldClone(pListsByName, now, playlist) {
			toClones = append(toClones, pListsByName[playlist])
		} else {
			log.Printf("not cloning %v", playlist)
		}
	}
	for _, toClone := range toClones {
		log.Printf("cloning %v", toClone.Name)
		err = clone(creds, now, toClone)
		if err != nil {
			return err
		}
	}
	return nil
}

func getAllPlaylists(creds db.PerUserCreds) ([]types.PlaylistResponseItem, error) {
	out := make([]types.PlaylistResponseItem, 0)
	do := func(offset, limit int) (*types.Paging, error) {
		resp, err := playlists(creds, offset, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Items...)
		return &resp.Paging, nil
	}
	err := getAll(do)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func getCared(cared []string, items []types.PlaylistResponseItem) []types.PlaylistResponseItem {
	out := make([]types.PlaylistResponseItem, len(cared))
	for _, e := range items {
		for _, c := range cared {
			if e.Name == c {
				out = append(out, e)
			}
		}
	}
	return out
}

func lessThanEqualMonday(t time.Time) time.Time {
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, -1)
	}
	return t.Truncate(24 * time.Hour)
}

func cloneName(name string, asOf time.Time) string {
	stamp := lessThanEqualMonday(asOf).Format("1/2/2006")
	return fmt.Sprintf("%v %v", name, stamp)
}

func shouldClone(playlists map[string]types.PlaylistResponseItem, asOf time.Time, name string) bool {
	if _, ok := playlists[name]; !ok {
		log.Print(name, " not found in original playlists list ", playlists)
		return false
	}
	_, ok := playlists[cloneName(name, asOf)]
	return !ok
}

func post(creds db.PerUserCreds, url string, jsonBody map[string]interface{}, v interface{}) error {
	log.Println("posting to ", url)
	bs, err := json.Marshal(jsonBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bs))
	addAuth(creds, req)
	req.Header["Accept"] = []string{"application/json"}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

const maxTracksPerPost = 100

func clone(creds db.PerUserCreds, asOf time.Time, playlist types.PlaylistResponseItem) error {
	var newp types.NewPlaylistResponse
	newPlaylistName := cloneName(playlist.Name, asOf)
	url := fmt.Sprintf("https://api.spotify.com/v1/users/%v/playlists", creds.Id)
	log.Println("mean to post to ", url, " and create ", newPlaylistName)
	err := post(creds, url, map[string]interface{}{"name": newPlaylistName, "public": false}, &newp)
	if err != nil {
		return err
	}
	uris, err := getAllTracks(creds, playlist)
	if err != nil {
		return err
	}
	log.Printf("trying to add %v songs", len(uris))
	url = fmt.Sprintf("https://api.spotify.com/v1/users/%v/playlists/%v/tracks", creds.Id, newp.Id)
	for i := 0; i < len(uris); i = i + maxTracksPerPost {
		end := i + maxTracksPerPost
		if end > len(uris) {
			end = len(uris)
		}
		err = post(creds, url, map[string]interface{}{"uris": uris[i:end]}, nil)
		if err != nil {
			return err
		}
	}
	log.Println("supposedly cloned")
	return nil
}
