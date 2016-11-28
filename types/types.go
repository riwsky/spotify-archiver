package types

type Paging struct {
	Limit  int
	Next   string
	Offset *int
	Total  int
}

type PlaylistResponseItem struct {
	Name   string
	Tracks struct {
		HREF  string
		Total int
	}
	Id    string
	Owner struct {
		Id string
	}
}

type PlaylistResponse struct {
	Paging
	Items []PlaylistResponseItem
}

type TrackResponse struct {
	Paging
	Items []struct {
		Track struct {
			URI string
		}
	}
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type NewPlaylistResponse struct {
	Id string
}
