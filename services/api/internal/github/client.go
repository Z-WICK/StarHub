package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL = "https://api.github.com"
)

var ErrNotFound = errors.New("not found")

type Client struct {
	httpClient *http.Client
}

type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
}

type Repo struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	FullName        string     `json:"full_name"`
	Private         bool       `json:"private"`
	HTMLURL         string     `json:"html_url"`
	Description     string     `json:"description"`
	Language        string     `json:"language"`
	StargazersCount int        `json:"stargazers_count"`
	PushedAt        *time.Time `json:"pushed_at"`
	Owner           struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type StarItem struct {
	StarredAt  time.Time `json:"starred_at"`
	Repository Repo      `json:"repo"`
}

type Pagination struct {
	NextPage int
	ETag     string
}

type Readme struct {
	Content string `json:"content"`
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) GetAuthenticatedUser(ctx context.Context, token string) (User, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/user", nil)
	if err != nil {
		return User{}, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return User{}, fmt.Errorf("github user request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return User{}, fmt.Errorf("github user request failed with status %d", response.StatusCode)
	}

	var user User
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		return User{}, fmt.Errorf("decode github user: %w", err)
	}
	return user, nil
}

func (c *Client) ListStarred(ctx context.Context, token string, page int, etag string) ([]StarItem, Pagination, bool, int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/user/starred?per_page=100&page=%d", baseURL, page), nil)
	if err != nil {
		return nil, Pagination{}, false, 0, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github.star+json")
	if etag != "" {
		request.Header.Set("If-None-Match", etag)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, Pagination{}, false, 0, fmt.Errorf("github starred request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified {
		return nil, Pagination{ETag: response.Header.Get("ETag")}, false, 0, nil
	}
	if response.StatusCode == http.StatusForbidden {
		retryAfter, _ := strconv.Atoi(response.Header.Get("Retry-After"))
		if retryAfter <= 0 {
			retryAfter = 60
		}
		return nil, Pagination{}, true, retryAfter, nil
	}
	if response.StatusCode >= http.StatusBadRequest {
		return nil, Pagination{}, false, 0, fmt.Errorf("github starred request failed with status %d", response.StatusCode)
	}

	items := make([]StarItem, 0)
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		return nil, Pagination{}, false, 0, fmt.Errorf("decode starred list: %w", err)
	}

	nextPage := parseNextPage(response.Header.Get("Link"))
	return items, Pagination{NextPage: nextPage, ETag: response.Header.Get("ETag")}, false, 0, nil
}

func (c *Client) GetReadme(ctx context.Context, token string, owner string, repo string) (Readme, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/repos/%s/%s/readme", baseURL, url.PathEscape(owner), url.PathEscape(repo)),
		nil,
	)
	if err != nil {
		return Readme{}, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Readme{}, fmt.Errorf("github readme request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return Readme{}, ErrNotFound
	}
	if response.StatusCode >= http.StatusBadRequest {
		return Readme{}, fmt.Errorf("github readme request failed with status %d", response.StatusCode)
	}

	var readme Readme
	if err := json.NewDecoder(response.Body).Decode(&readme); err != nil {
		return Readme{}, fmt.Errorf("decode github readme: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(readme.Content, "\n", ""))
	if err != nil {
		return Readme{}, fmt.Errorf("decode github readme content: %w", err)
	}
	readme.Content = string(decoded)

	return readme, nil
}

func parseNextPage(link string) int {
	if link == "" {
		return 0
	}
	parts := strings.Split(link, ",")
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if !strings.Contains(p, "rel=\"next\"") {
			continue
		}
		start := strings.Index(p, "page=")
		if start == -1 {
			continue
		}
		start += len("page=")
		end := start
		for end < len(p) && p[end] >= '0' && p[end] <= '9' {
			end++
		}
		if end == start {
			continue
		}
		page, err := strconv.Atoi(p[start:end])
		if err == nil {
			return page
		}
	}
	return 0
}
