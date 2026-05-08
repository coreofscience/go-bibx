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

var (
	workFields     []string
	workFieldsOnce sync.Once
)

func getWorkFields() []string {
	workFieldsOnce.Do(func() {
		workFields = jsonFields(Work{})
	})
	return workFields
}

// ListRecentArticles implements [Client]
func (c *RestyClient) ListRecentArticles(ctx context.Context, query string, limit int) ([]Work, error) {
	selectFields := getWorkFields()
	queryFilter := fmt.Sprintf(
		"title_and_abstract.search:%s",
		strings.ReplaceAll(query, " ", "+"),
	)
	filterParts := []string{
		queryFilter,
		"type:types/article",
		"cited_by_count:>1",
		"has_abstract:true",
	}

	maxPages := int(math.Ceil(float64(limit) / MaxWorksPerPage))
	pages := make([]int, maxPages)
	for i := range maxPages {
		pages[i] = i + 1
	}

	works, err := fetchParallel(ctx, pages, func(ctx context.Context, page int) (*WorksResponse, error) {
		queryParams := map[string]string{
			"select":   strings.Join(selectFields, ","),
			"filter":   strings.Join(filterParts, ","),
			"sort":     "publication_year:desc",
			"per_page": fmt.Sprintf("%d", MaxWorksPerPage),
			"page":     fmt.Sprintf("%d", page),
		}
		return c.fetchWorks(ctx, queryParams)
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching works in parallel: %w", err)
	}

	if len(works) > limit {
		works = works[:limit]
	}
	return works, nil
}

// ListArticlesByIDs implements [Client]
func (c *RestyClient) ListArticlesByIDs(ctx context.Context, ids []string) ([]Work, error) {
	if len(ids) == 0 {
		return []Work{}, nil
	}

	selectFields := getWorkFields()
	idChunks := chunks(ids, MaxIdsPerRequest)

	return fetchParallel(ctx, idChunks, func(ctx context.Context, idChunk []string) (*WorksResponse, error) {
		joinedIDs := strings.Join(idChunk, "|")
		queryParams := map[string]string{
			"select":   strings.Join(selectFields, ","),
			"filter":   fmt.Sprintf("ids.openalex:%s,type:types/article", joinedIDs),
			"per_page": fmt.Sprintf("%d", MaxIdsPerRequest),
		}
		return c.fetchWorks(ctx, queryParams)
	})
}

func fetchParallel[I any](ctx context.Context, inputs []I, fetch func(context.Context, I) (*WorksResponse, error)) ([]Work, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	group, ctx := errgroup.WithContext(ctx)
	inputChan := make(chan I)
	responses := make(chan *WorksResponse)

	concurrency := min(MaxConcurrentRequests, len(inputs))
	slog.Debug("fetching in parallel", "numInputs", len(inputs), "concurrency", concurrency)

	for range concurrency {
		group.Go(func() error {
			for input := range inputChan {
				res, err := fetch(ctx, input)
				if err != nil {
					return err
				}
				if res != nil {
					select {
					case responses <- res:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return nil
		})
	}

	waitGroup := sync.WaitGroup{}
	waitGroup.Go(func() {
		defer close(inputChan)
		for _, input := range inputs {
			select {
			case inputChan <- input:
			case <-ctx.Done():
				return
			}
		}
	})

	var works []Work
	waitGroup.Go(func() {
		for res := range responses {
			if res != nil {
				works = append(works, res.Works...)
			}
		}
	})

	err := group.Wait()
	close(responses)
	if err != nil {
		return nil, err
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
	for field := range tType.Fields() {
		field := field
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
