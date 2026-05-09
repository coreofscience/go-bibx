package openalex_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coreofscience/go-bibx/internal/clients/openalex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func TestListArticlesByIDs(t *testing.T) {
	t.Run("empty ids", func(t *testing.T) {
		client := openalex.NewRestyClient()
		works, err := client.ListArticlesByIDs(context.Background(), []string{})
		assert.NoError(t, err)
		assert.Empty(t, works)
	})

	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filter := r.URL.Query().Get("filter")
			assert.Contains(t, filter, "ids.openalex:W1|W2")

			resp := openalex.WorksResponse{
				Works: []openalex.Work{
					{ID: "W1", Title: new("Title 1")},
					{ID: "W2", Title: new("Title 2")},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer server.Close()

		client := openalex.NewRestyClient(openalex.WithBaseURL(server.URL))
		works, err := client.ListArticlesByIDs(context.Background(), []string{"W1", "W2"})

		assert.NoError(t, err)
		if assert.Len(t, works, 2) {
			assert.Equal(t, "W1", works[0].ID)
			assert.Equal(t, "W2", works[1].ID)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := openalex.NewRestyClient(
			openalex.WithBaseURL(server.URL),
			openalex.WithHTTPClient(resty.New()), // Disable retries
		)
		_, err := client.ListArticlesByIDs(context.Background(), []string{"W1"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500 Internal Server Error")
	})

	t.Run("multiple chunks", func(t *testing.T) {
		// MaxIdsPerRequest is 80, so 85 ids should result in 2 chunks
		ids := make([]string, 85)
		for i := range 85 {
			ids[i] = fmt.Sprintf("W%d", i)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filter := r.URL.Query().Get("filter")
			// Each chunk should be handled
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(filter, "W0") {
				resp := openalex.WorksResponse{Works: make([]openalex.Work, 80)}
				err := json.NewEncoder(w).Encode(resp)
				require.NoError(t, err)
			} else {
				resp := openalex.WorksResponse{Works: make([]openalex.Work, 5)}
				err := json.NewEncoder(w).Encode(resp)
				require.NoError(t, err)
			}
		}))
		defer server.Close()

		client := openalex.NewRestyClient(openalex.WithBaseURL(server.URL))
		works, err := client.ListArticlesByIDs(context.Background(), ids)

		assert.NoError(t, err)
		assert.Len(t, works, 85)
	})
}

func TestListRecentArticles(t *testing.T) {
	t.Run("successful fetch with pagination", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			resp := openalex.WorksResponse{
				Works: []openalex.Work{
					{ID: fmt.Sprintf("W-%s", page), Title: new(fmt.Sprintf("Title %s", page))},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer server.Close()

		client := openalex.NewRestyClient(openalex.WithBaseURL(server.URL))
		// MaxWorksPerPage is 200, so limit 250 should request 2 pages
		works, err := client.ListRecentArticles(context.Background(), "test query", 250)

		assert.NoError(t, err)
		// Since our mock returns 1 work per page, we expect 2 works total
		assert.Len(t, works, 2)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client := openalex.NewRestyClient(
			openalex.WithBaseURL(server.URL),
			openalex.WithHTTPClient(resty.New()), // Disable retries
		)
		_, err := client.ListRecentArticles(context.Background(), "query", 10)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "503 Service Unavailable")
	})
}
