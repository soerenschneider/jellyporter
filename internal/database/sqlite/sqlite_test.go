package sqlite

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/jellyporter/internal/jellyfin"
)

func TestSQLiteQueue_GetUnwatchedMovies(t *testing.T) {
	type fields struct {
		db *SQLiteJellyDb
	}
	type movies map[string][]jellyfin.Item
	type args struct {
		ctx    context.Context
		server string
	}
	tests := []struct {
		name    string
		fields  fields
		input   movies
		args    args
		want    []ItemWithUpdatedUserData
		wantErr bool
	}{
		{
			name: "Three servers, no playback yet",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": {
					{
						Name:     "The Matrix",
						ServerID: "dd",
						ID:       "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": {
					{
						Name:     "The Matrix",
						ServerID: "ez",
						ID:       "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": {
					{
						Name:     "The Matrix",
						ServerID: "pt",
						ID:       "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
		{
			name: "Three servers, playback on one server, not the one running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": {
					{
						Name:     "The Matrix",
						ServerID: "dd",
						ID:       "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": {
					{
						Name:     "The Matrix",
						ServerID: "ez",
						ID:       "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": {
					{
						Name:     "The Matrix",
						ServerID: "pt",
						ID:       "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want: []ItemWithUpdatedUserData{
				{
					Name:                 "The Matrix",
					LocalID:              "1",
					WatchedDate:          time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()).Unix(),
					WatchedProgress:      0.5,
					WatchedPositionTicks: 12874613523,
				},
			},
			wantErr: false,
		},
		{
			name: "Three servers, playback on a single server, the one running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": {
					{
						Name:     "The Matrix",
						ServerID: "dd",
						ID:       "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": {
					{
						Name:     "The Matrix",
						ServerID: "ez",
						ID:       "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": {
					{
						Name:     "The Matrix",
						ServerID: "pt",
						ID:       "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
		{
			name: "Three servers, playback on all servers, one ahead, *not* the server running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": {
					{
						Name:     "The Matrix",
						ServerID: "dd",
						ID:       "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
							IsFavorite:            false,
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": {
					{
						Name:     "The Matrix",
						ServerID: "ez",
						ID:       "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 07, 15, 15, 0, 0, 0, time.Now().Location()),
							IsFavorite:            true,
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": {
					{
						Name:     "The Matrix",
						ServerID: "pt",
						ID:       "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
							IsFavorite:            false,
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want: []ItemWithUpdatedUserData{
				{
					Name:                 "The Matrix",
					LocalID:              "1",
					WatchedDate:          time.Date(2025, 07, 15, 15, 0, 0, 0, time.Now().Location()).Unix(),
					WatchedProgress:      0.5,
					WatchedPositionTicks: 12874613523,
					IsFavorite:           true,
				},
			},
			wantErr: false,
		},
		{
			name: "Three servers, playback on all servers, one ahead, on the server running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": {
					{
						Name:     "The Matrix",
						ServerID: "dd",
						ID:       "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 07, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": {
					{
						Name:     "The Matrix",
						ServerID: "ez",
						ID:       "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": {
					{
						Name:     "The Matrix",
						ServerID: "pt",
						ID:       "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
		{
			name: "Three servers, identical non-zero playback on all servers",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": {
					{
						Name:     "The Matrix",
						ServerID: "dd",
						ID:       "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": {
					{
						Name:     "The Matrix",
						ServerID: "ez",
						ID:       "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": {
					{
						Name:     "The Matrix",
						ServerID: "pt",
						ID:       "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.fields.db
			for key, movie := range tt.input {
				if err := q.InsertMovies(t.Context(), key, movie); err != nil {
					log.Fatal().Err(err).Msgf("could not insert movie")
				}
			}
			got, err := q.GetMoviesWithUpdatedUserData(tt.args.ctx, tt.args.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMoviesWithUpdatedUserData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMoviesWithUpdatedUserData() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLiteQueue_GetUnwatchedEpisodes(t *testing.T) {
	type fields struct {
		db *SQLiteJellyDb
	}
	type movies map[string][]jellyfin.Item
	type args struct {
		ctx    context.Context
		server string
	}
	tests := []struct {
		name    string
		fields  fields
		input   movies
		args    args
		want    []ItemWithUpdatedUserData
		wantErr bool
	}{
		{
			name: "Three servers, no playback yet",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "dd",
						ID:         "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "ez",
						ID:         "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "pt",
						ID:         "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
		{
			name: "Three servers, playback on one server, not the one running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "dd",
						ID:         "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "ez",
						ID:         "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
							IsFavorite:            true,
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "pt",
						ID:         "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want: []ItemWithUpdatedUserData{
				{
					Name:                 "Episode I",
					SeriesName:           "Black Mirror",
					LocalID:              "1",
					WatchedDate:          time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()).Unix(),
					WatchedProgress:      0.5,
					WatchedPositionTicks: 12874613523,
					IsFavorite:           true,
				},
			},
			wantErr: false,
		},
		{
			name: "Three servers, playback on a single server, the one running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "dd",
						ID:         "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "ez",
						ID:         "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "pt",
						ID:         "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Time{},
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
		{
			name: "Three servers, playback on all servers, one ahead, *not* the server running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "dd",
						ID:         "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "ez",
						ID:         "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 07, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "pt",
						ID:         "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want: []ItemWithUpdatedUserData{
				{
					Name:                 "Episode I",
					SeriesName:           "Black Mirror",
					LocalID:              "1",
					WatchedDate:          time.Date(2025, 07, 15, 15, 0, 0, 0, time.Now().Location()).Unix(),
					WatchedProgress:      0.5,
					WatchedPositionTicks: 12874613523,
				},
			},
			wantErr: false,
		},
		{
			name: "Three servers, playback on all servers, one ahead, on the server running the query",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "dd",
						ID:         "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 07, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "ez",
						ID:         "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "pt",
						ID:         "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
		{
			name: "Three servers, identical non-zero playback on all servers",
			fields: fields{
				db: MustNew(""),
			},
			input: map[string][]jellyfin.Item{
				"dd": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "dd",
						ID:         "1",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"ez": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "ez",
						ID:         "2",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 12874613523,
							PlayedPercentage:      0.5,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
				"pt": []jellyfin.Item{
					{
						Name:       "Episode I",
						SeriesName: "Black Mirror",
						SeasonName: "S01",
						ServerID:   "pt",
						ID:         "3",
						UserData: jellyfin.UserData{
							PlaybackPositionTicks: 0,
							PlayedPercentage:      0,
							LastPlayedDate:        time.Date(2025, 06, 15, 15, 0, 0, 0, time.Now().Location()),
						},
						ProviderIDs: jellyfin.ProviderIDs{
							IMDB: "133093",
							TMDB: "603",
						},
						Runtime: 5000,
					},
				},
			},
			args: args{
				ctx:    t.Context(),
				server: "dd",
			},
			want:    []ItemWithUpdatedUserData{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.fields.db
			for key, episode := range tt.input {
				if err := q.InsertEpisodes(t.Context(), key, episode); err != nil {
					log.Fatal().Err(err).Msgf("could not insert episode")
				}
			}

			got, err := q.GetEpisodesWithUpdatedUserData(tt.args.ctx, tt.args.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEpisodesWithUpdatedUserData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEpisodesWithUpdatedUserData() got = %v, want %v", got, tt.want)
			}
		})
	}
}
