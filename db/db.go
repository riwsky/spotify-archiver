package db

import (
	"encoding/json"
	"github.com/riwsky/spotify-archiver/types"
	"os"
)

type PerUserCreds struct {
	AccessToken  string
	RefreshToken string
	Code         string
	Id           string
}

type DB struct {
	UserCreds map[string]PerUserCreds
}

const cacheLocation = "/tmp/cache.json"

func Load() (*DB, error) {
	file, err := os.Open(cacheLocation)
	defer file.Close()
	if err != nil {
		if os.IsNotExist(err) {
			return &DB{UserCreds: map[string]PerUserCreds{}}, nil
		}
		return nil, err
	}
	var s DB
	err = json.NewDecoder(file).Decode(&s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) Add(id, code string, auth types.AuthResponse) error {
	creds := PerUserCreds{
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		Id:           id,
		Code:         code,
	}
	d.UserCreds[id] = creds
	return nil
}

func (d DB) Save() error {
	file, err := os.Create(cacheLocation)
	defer file.Close()
	if err != nil {
		return err
	}
	bs, err := json.Marshal(d)
	if err != nil {
		return err
	}
	_, err = file.Write(bs)
	if err != nil {
		return err
	}
	return nil
}
