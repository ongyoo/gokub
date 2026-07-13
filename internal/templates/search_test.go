package templates

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestSearchTemplates(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/search/repositories" || !strings.Contains(request.URL.Query().Get("q"), "topic:gokub-template api") {
			t.Fatalf("unexpected request: %s", request.URL.String())
		}
		if request.Header.Get("Authorization") != "Bearer token" || request.Header.Get("User-Agent") != "gokub-cli" {
			t.Fatalf("missing GitHub headers: %v", request.Header)
		}
		body := `{"items":[{"full_name":"team/api-template","description":"API starter","stargazers_count":42,"default_branch":"main","html_url":"https://github.com/team/api-template"},{"full_name":"team/archived","archived":true}]}`
		return response(http.StatusOK, body), nil
	})}

	results, err := Search(SearchOptions{Query: "api", Limit: 5, Token: "token", Client: client})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Repository != "team/api-template" || results[0].Stars != 42 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestSearchTemplatesReportsAPIError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusForbidden, `{"message":"rate limit exceeded"}`), nil
	})}
	_, err := Search(SearchOptions{Client: client})
	if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
