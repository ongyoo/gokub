package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SearchResult struct {
	Repository    string `json:"repository"`
	Description   string `json:"description"`
	Stars         int    `json:"stars"`
	DefaultBranch string `json:"default_branch"`
	URL           string `json:"url"`
}

type SearchOptions struct {
	Query   string
	Limit   int
	Token   string
	APIBase string
	Client  *http.Client
}

func Search(options SearchOptions) ([]SearchResult, error) {
	if options.Limit <= 0 {
		options.Limit = 10
	}
	if options.Limit > 50 {
		return nil, fmt.Errorf("template search limit cannot exceed 50")
	}
	if options.APIBase == "" {
		options.APIBase = "https://api.github.com"
	}
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 10 * time.Second}
	}
	endpoint, err := url.Parse(strings.TrimRight(options.APIBase, "/") + "/search/repositories")
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	search := "topic:gokub-template"
	if value := strings.TrimSpace(options.Query); value != "" {
		search += " " + value
	}
	query.Set("q", search)
	query.Set("sort", "stars")
	query.Set("order", "desc")
	query.Set("per_page", strconv.Itoa(options.Limit))
	endpoint.RawQuery = query.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "gokub-cli")
	if options.Token != "" {
		request.Header.Set("Authorization", "Bearer "+options.Token)
	}
	response, err := options.Client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("search GitHub templates: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		var failure struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if failure.Message == "" {
			failure.Message = response.Status
		}
		return nil, fmt.Errorf("GitHub template search failed: %s", failure.Message)
	}
	var payload struct {
		Items []struct {
			FullName      string `json:"full_name"`
			Description   string `json:"description"`
			Stars         int    `json:"stargazers_count"`
			DefaultBranch string `json:"default_branch"`
			HTMLURL       string `json:"html_url"`
			Archived      bool   `json:"archived"`
			Disabled      bool   `json:"disabled"`
		} `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode GitHub template search: %w", err)
	}
	results := make([]SearchResult, 0, len(payload.Items))
	for _, item := range payload.Items {
		if item.Archived || item.Disabled {
			continue
		}
		if _, _, err := normalizeRepository(item.FullName, false); err != nil {
			continue
		}
		results = append(results, SearchResult{
			Repository: item.FullName, Description: item.Description, Stars: item.Stars,
			DefaultBranch: item.DefaultBranch, URL: item.HTMLURL,
		})
	}
	return results, nil
}
