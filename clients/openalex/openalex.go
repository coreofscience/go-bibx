package openalex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"resty.dev/v3"
)

const (
	MaxWorksPerPage       = 200
	MaxIdsPerRequest      = 80
	MaxConcurrentRequests = 4
)

// Client is the interface for the OpenAlex client.
type Client interface {
	// ListRecentArticles lists recent articles based on the given query and limit.
	ListRecentArticles(ctx context.Context, query string, limit int) ([]Work, error)

	// ListArticlesByIDs lists articles by their IDs.
	ListArticlesByIDs(ctx context.Context, ids []string) ([]Work, error)
}

// RestyClient is the concrete implementation of the OpenAlex client using resty.
type RestyClient struct {
	baseURL     string
	client      *resty.Client
	baseHeaders map[string]string
}

// OpenAlexClientOption is a function that configures the RestyClient.
type OpenAlexClientOption func(*RestyClient)

// WithBaseURL sets the base URL for the OpenAlex client.
func WithBaseURL(url string) OpenAlexClientOption {
	return func(c *RestyClient) {
		c.baseURL = url
	}
}

// WithHTTPClient sets the HTTP client for the OpenAlex client.
func WithHTTPClient(client *resty.Client) OpenAlexClientOption {
	return func(c *RestyClient) {
		c.client = client
	}
}

// WithHeader sets a header for the OpenAlex client.
func WithHeader(key string, value string) OpenAlexClientOption {
	return func(c *RestyClient) {
		if c.baseHeaders == nil {
			c.baseHeaders = make(map[string]string)
		}
		c.baseHeaders[key] = value
	}
}

// WithEmail sets the email for the OpenAlex client.
func WithEmail(email string) OpenAlexClientOption {
	return func(c *RestyClient) {
		if c.baseHeaders == nil {
			c.baseHeaders = make(map[string]string)
		}
		c.baseHeaders["User-Agent"] = fmt.Sprintf("Go/resty/go-bibx mailto:%s", email)
	}
}

// NewRestyClient creates a new OpenAlex client with the given options.
func NewRestyClient(options ...OpenAlexClientOption) *RestyClient {
	c := &RestyClient{
		baseURL: "https://api.openalex.org",
		client:  resty.New(),
		baseHeaders: map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
			"User-Agent":   fmt.Sprintf("Go/resty/go-bibx mailto:%s", "technology@coreofscience.org"),
		},
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// ListRecentArticles implements [Client]
func (c *RestyClient) ListRecentArticles(ctx context.Context, query string, limit int) ([]Work, error) {
	selectFields := jsonFields(Work{})
	queryFilter := fmt.Sprintf(
		"title_and_abstract.search:%s",
		strings.ReplaceAll(query, " ", "+"),
	)
	filterParts := []string{
		queryFilter,
		"type:types/article",
		"cited_by_count:>1",
	}

	group, ctx := errgroup.WithContext(ctx)
	pages := make(chan int)
	responses := make(chan *WorksResponse)

	maxPages := int(math.Ceil(float64(limit) / MaxWorksPerPage))
	concurrency := min(MaxConcurrentRequests, maxPages)
	slog.Debug("fetching pages with concurrency", "maxPages", maxPages, "concurrency", concurrency)

	for i := range concurrency {
		goroutineIndex := i
		group.Go(func() error {
			for page := range pages {
				slog.Debug("fetching page from goroutine", "page", page, "goroutine", goroutineIndex)
				queryParams := map[string]string{
					"select":   strings.Join(selectFields, ","),
					"filter":   strings.Join(filterParts, ","),
					"sort":     "publication_year:desc",
					"per_page": fmt.Sprintf("%d", MaxWorksPerPage),
					"page":     fmt.Sprintf("%d", page),
				}
				response, err := c.client.R().
					SetContext(ctx).
					SetHeaders(c.baseHeaders).
					SetQueryParams(queryParams).
					SetResult(&WorksResponse{}).
					Get(fmt.Sprintf("%s/works", c.baseURL))
				if err != nil {
					return fmt.Errorf("error fetching works: %w", err)
				}
				if response.IsError() {
					slog.Debug("error response", "status", response.Status(), "body", response.String())
					return fmt.Errorf("error fetching works: %s", response.Status())
				}
				result := response.Result().(*WorksResponse)
				if result == nil {
					return errors.New("error parsing works response")
				}
				slog.Debug("fetched page", "page", page, "numWorks", len(result.Works))
				if len(result.Works) == 0 {
					slog.Debug("no more works, stopping", "page", page)
					continue
				}
				slog.Debug("sending response", "numWorks", len(result.Works))
				responses <- result
			}
			return nil
		})
	}

	for page := 0; page < int(maxPages); page++ {
		pages <- page + 1
	}
	close(pages)

	works := make([]Work, 0, limit)
	waitGroup := sync.WaitGroup{}
	waitGroup.Go(func() {
		slog.Debug("starting response collector")
		for res := range responses {
			if res != nil {
				slog.Debug("received response", "numWorks", len(res.Works))
				works = append(works, res.Works...)
			}
		}
		slog.Debug("response collector done", "totalWorks", len(works))
	})

	err := group.Wait()
	close(responses)
	if err != nil {
		return nil, fmt.Errorf("error fetching works in parallel: %w", err)
	}
	waitGroup.Wait()
	slog.Debug("all done", "totalWorks", len(works))
	return works, nil
}

// ListArticlesByIDs implements [Client]
func (c *RestyClient) ListArticlesByIDs(ctx context.Context, ids []string) ([]Work, error) {
	if len(ids) == 0 {
		return []Work{}, nil
	}

	selectFields := jsonFields(Work{})
	idChunks := chunks(ids, MaxIdsPerRequest)

	group, ctx := errgroup.WithContext(ctx)
	chunksChan := make(chan []string)
	responses := make(chan *WorksResponse)

	concurrency := min(MaxConcurrentRequests, len(idChunks))
	slog.Debug("fetching pages with concurrency", "numChunks", len(idChunks), "concurrency", concurrency)

	for i := range concurrency {
		goroutineIndex := i
		group.Go(func() error {
			for idChunk := range chunksChan {
				slog.Debug("fetching chunk from goroutine", "numIds", len(idChunk), "goroutine", goroutineIndex)
				joinedIDs := strings.Join(idChunk, "|")
				queryParams := map[string]string{
					"select":   strings.Join(selectFields, ","),
					"filter":   fmt.Sprintf("ids.openalex:%s,type:types/article", joinedIDs),
					"per_page": fmt.Sprintf("%d", MaxIdsPerRequest),
				}
				response, err := c.fetchWorks(ctx, queryParams)
				if err != nil {
					return fmt.Errorf("error fetching works: %w", err)
				}
				if response == nil || len(response.Works) == 0 {
					continue
				}
				slog.Debug("fetched chunk", "numWorks", len(response.Works))
				responses <- response
			}
			return nil
		})
	}

	waitGroup := sync.WaitGroup{}

	waitGroup.Go(func() {
		slog.Debug("sending id chunks to workers", "numChunks", len(idChunks))
		for _, idChunk := range idChunks {
			chunksChan <- idChunk
		}
		close(chunksChan)
	})

	works := make([]Work, 0)
	waitGroup.Go(func() {
		slog.Debug("starting response collector")
		for res := range responses {
			if res != nil {
				slog.Debug("received response", "numWorks", len(res.Works))
				works = append(works, res.Works...)
			}
		}
	})

	err := group.Wait()
	close(responses)
	if err != nil {
		return nil, fmt.Errorf("error fetching works in parallel: %w", err)
	}

	waitGroup.Wait()

	return works, nil
}

func (c *RestyClient) fetchWorks(ctx context.Context, queryParams map[string]string) (*WorksResponse, error) {
	response, err := c.client.R().
		SetContext(ctx).
		SetHeaders(c.baseHeaders).
		SetQueryParams(queryParams).
		SetResult(&WorksResponse{}).
		Get(fmt.Sprintf("%s/works", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("error fetching works: %w", err)
	}
	if response.IsError() {
		slog.Debug("error response", "status", response.Status(), "body", response.String())
		return nil, fmt.Errorf("error fetching works: %s", response.Status())
	}
	result := response.Result().(*WorksResponse)
	if result == nil {
		return nil, errors.New("error parsing works response")
	}
	return result, nil
}

// list all the JSON fields for an arbitrary struct
func jsonFields(t any) []string {
	var fields []string
	v := reflect.ValueOf(t)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fields
	}
	tType := v.Type()
	for i := 0; i < tType.NumField(); i++ {
		field := tType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			fields = append(fields, jsonTag)
		}
	}
	return fields
}

// chunks splits a slice into chunks of the specified size
func chunks[T any](slice []T, chunkSize int) [][]T {
	numChunks := (len(slice) + chunkSize - 1) / chunkSize
	chunked := make([][]T, 0, numChunks)
	for i := 0; i < len(slice); i += chunkSize {
		end := min(i+chunkSize, len(slice))
		chunked = append(chunked, slice[i:end])
	}
	return chunked
}
