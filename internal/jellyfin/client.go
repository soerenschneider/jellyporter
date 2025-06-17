package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/soerenschneider/jellyporter/internal/metrics"
)

type ItemType string
type SortFields string
type SortOrder string

const (
	ItemEpisode         ItemType   = "Episode"
	ItemMovie           ItemType   = "Movie"
	SortFieldDatePlayed SortFields = "DatePlayed"
	SortOrderAscending  SortOrder  = "Ascending"
	SortOrderDescending SortOrder  = "Descending"
)

var (
	defaultClient = newConfiguredClient()
	validation    = validator.New()
)

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client

	userName string
	userId   string

	mutex sync.Mutex
}

func NewJellyfinClient(baseURL, apiKey, userName string) *Client {
	return &Client{
		baseURL:  baseURL,
		apiKey:   apiKey,
		userName: userName,
		client:   defaultClient,
	}
}

type ItemQueryOpts struct {
	Limit      int `validate:"gte=25,lte=1000"`
	Since      *time.Time
	StartIndex int `validate:"gte=0"`
	SortBy     SortFields
	SortOrder  SortOrder
	Type       ItemType `validate:"required,oneof=Movie Episode"`
}

func (o ItemQueryOpts) IsDelta() bool {
	return o.Since != nil
}

func (j *Client) GetItems(ctx context.Context, userID string, opts ItemQueryOpts) (*ItemsResponse, error) {
	if err := validation.Struct(opts); err != nil {
		return nil, fmt.Errorf("validation of query opts failed: %w", err)
	}

	var allMovies []Item
	startIndex := opts.StartIndex

	reachedEnd := false
	for !reachedEnd {
		params := url.Values{}
		params.Set("IncludeItemTypes", string(opts.Type))
		params.Set("Recursive", "true")
		params.Set("Fields", "ProviderIds")
		params.Set("Limit", fmt.Sprintf("%d", opts.Limit))
		params.Set("StartIndex", fmt.Sprintf("%d", startIndex))
		params.Set("EnableTotalRecordCount", "true")

		if opts.SortBy != "" {
			params.Set("SortBy", string(opts.SortBy))
		}
		if opts.SortOrder != "" {
			params.Set("SortOrder", string(opts.SortOrder))
		}

		endpoint := fmt.Sprintf("/Users/%s/Items?%s", userID, params.Encode())

		data, err := j.makeRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		var response ItemsResponse
		if err := json.Unmarshal(data, &response); err != nil {
			return nil, err
		}

		exceededTimeFilter := false
		if opts.Since != nil {
			lastEpisode, found := lastElement[Item](response.Items)
			if found && lastEpisode.UserData.LastPlayedDate.Before(*opts.Since) {
				exceededTimeFilter = true
				for _, item := range response.Items {
					if item.UserData.LastPlayedDate.After(*opts.Since) {
						allMovies = append(allMovies, item)
					}
				}
			} else {
				// add all items as they all seem to be within the time limit
				allMovies = append(allMovies, response.Items...)
			}
		} else {
			allMovies = append(allMovies, response.Items...)
		}

		if len(response.Items) < opts.Limit || startIndex+len(response.Items) >= response.TotalRecordCount || exceededTimeFilter {
			reachedEnd = true
		}

		startIndex += opts.Limit
	}

	return &ItemsResponse{
		Items:            allMovies,
		TotalRecordCount: len(allMovies),
		StartIndex:       0,
	}, nil
}

func (j *Client) UpdateUserData(ctx context.Context, userID, itemID string, userData UserDataUpdate) error {
	endpoint := fmt.Sprintf("/Users/%s/Items/%s/UserData", userID, itemID)

	jsonData, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	_, err = j.makeRequest(ctx, http.MethodPost, endpoint, jsonData)
	return err
}

func (j *Client) MarkWatched(ctx context.Context, userID, itemID string) error {
	endpoint := fmt.Sprintf("/Users/%s/PlayedItems/%s", userID, itemID)

	_, err := j.makeRequest(ctx, http.MethodPost, endpoint, nil)
	return err
}

func (j *Client) GetUserId(ctx context.Context) (string, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.userId != "" {
		return j.userId, nil
	}

	user, err := j.GetUser(ctx, j.userName)
	if err != nil {
		return "", err
	}

	j.userId = user.ID
	return j.userId, nil
}

func (j *Client) GetUser(ctx context.Context, name string) (User, error) {
	users, err := j.GetUsers(ctx)
	if err != nil {
		return User{}, err
	}

	for _, user := range users {
		if user.Name == name {
			return user, nil
		}
	}

	return User{}, errors.New("user not found")
}

func (j *Client) GetUsers(ctx context.Context) ([]User, error) {
	data, err := j.makeRequest(ctx, "GET", "/Users", nil)
	if err != nil {
		return nil, err
	}

	var users []User
	err = json.Unmarshal(data, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// makeRequest performs an HTTP request and returns the response body
func (j *Client) makeRequest(ctx context.Context, method, endpoint string, body []byte) ([]byte, error) {
	metrics.RequestsTotal.Inc()
	start := time.Now()
	fullURL := fmt.Sprintf("%s%s", j.baseURL, endpoint)

	var req *http.Request

	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		metrics.RequestErrorsTotal.WithLabelValues("invalid_url", "unknown").Inc()
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, parsedURL.String(), bytes.NewBuffer(body))
		if err != nil {
			metrics.RequestErrorsTotal.WithLabelValues("request_error", parsedURL.Path).Inc()
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			metrics.RequestErrorsTotal.WithLabelValues("request_error", parsedURL.Path).Inc()
			return nil, err
		}
	}

	// Add API key to URL
	values := req.URL.Query()
	values.Add("api_key", j.apiKey)
	req.URL.RawQuery = values.Encode()

	resp, err := j.client.Do(req)
	if err != nil {
		metrics.RequestErrorsTotal.WithLabelValues("send_request_failed", parsedURL.Path).Inc()
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	metrics.RequestTime.WithLabelValues(parsedURL.Path, strconv.Itoa(resp.StatusCode)).Observe(time.Since(start).Seconds())

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		metrics.RequestErrorsTotal.WithLabelValues("invalid_status", parsedURL.Path).Inc()
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		metrics.RequestErrorsTotal.WithLabelValues("read_data", parsedURL.Path).Inc()
	}

	return data, err
}

func lastElement[T any](s []T) (T, bool) {
	var zero T
	if len(s) == 0 {
		return zero, false
	}
	return s[len(s)-1], true
}

func newConfiguredClient() *http.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 3

	// Set max backoff duration to 15s
	client.Backoff = retryablehttp.DefaultBackoff
	client.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		backoff := retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
		if backoff > 15*time.Second {
			return 15 * time.Second
		}
		return backoff
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client.HTTPClient = &http.Client{
		Transport: transport,
	}

	return client.HTTPClient
}
