package jellyfin

import "time"

type Item struct {
	Name        string      `json:"Name"`
	ServerID    string      `json:"ServerId"`
	ID          string      `json:"Id"`
	UserData    UserData    `json:"UserData"`
	ProviderIDs ProviderIDs `json:"ProviderIds"`
	Type        string      `json:"Type"`
	SeriesName  string      `json:"SeriesName"`
	SeriesId    string      `json:"SeriesId"`
	SeasonId    string      `json:"SeasonId"`
	SeasonName  string      `json:"SeasonName"`
	Runtime     int64       `json:"RunTimeTicks"`
}

type UserData struct {
	PlaybackPositionTicks int64     `json:"PlaybackPositionTicks"`
	PlayedPercentage      float64   `json:"PlayedPercentage"`
	PlayCount             int       `json:"PlayCount"`
	IsFavorite            bool      `json:"IsFavorite"`
	LastPlayedDate        time.Time `json:"LastPlayedDate"`
	Played                bool      `json:"Played"`
	Key                   string    `json:"Key"`
	ItemID                string    `json:"ItemId"`
}

type ProviderIDs struct {
	IMDB string `json:"Imdb,omitempty"`
	TMDB string `json:"Tmdb,omitempty"`
	TVDB string `json:"Tvdb,omitempty"`
}

type ItemsResponse struct {
	Items            []Item `json:"Items"`
	TotalRecordCount int    `json:"TotalRecordCount"`
	StartIndex       int    `json:"StartIndex"`
}

type WatcherOptions struct {
	Limit        int
	StartIndex   int
	WatchedAfter time.Time
}

type UserDataUpdate struct {
	IsFavorite            *bool     `json:"IsFavorite,omitempty"`
	PlaybackPositionTicks *int64    `json:"PlaybackPositionTicks"`
	PlayedPercentage      *float64  `json:"PlayedPercentage"`
	PlayCount             int       `json:"PlayCount"`
	LastPlayedDate        time.Time `json:"LastPlayedDate"`
	Played                bool      `json:"Played"`
	Key                   string    `json:"Key"`
	ItemID                string    `json:"ItemId"`
}

type User struct {
	Name     string `json:"Name"`
	ServerID string `json:"ServerId"`
	ID       string `json:"Id"`
}

type UsersResponse struct {
	Users []User `json:"Users,omitempty"`
}
