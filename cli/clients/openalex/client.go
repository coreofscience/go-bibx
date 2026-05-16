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
	"time"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/models"
	"golang.org/x/sync/errgroup"
	"resty.dev/v3"
)

const (
	MaxWorksPerPage       = 200
	MaxIDsPerRequest      = 50
	MaxConcurrentRequests = 4
)

// Client is the interface for the OpenAlex client.
type Client interface {
	// ListRecentArticles lists recent articles based on the given query and limit.
	ListRecentArticles(ctx context.Context, query string, limit int) ([]*Work, error)

	// ListArticlesByIDs lists articles by their IDs.
	ListArticlesByIDs(ctx context.Context, ids []string) ([]*Work, error)
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
	restyClient := resty.New().
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second).
		AddRetryConditions(func(r *resty.Response, err error) bool {
			return r.StatusCode() == 429 || r.StatusCode() >= 500
		})
	c := &RestyClient{
		baseURL: "https://api.openalex.org",
		client:  restyClient,
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
func (c *RestyClient) ListRecentArticles(ctx context.Context, query string, limit int) ([]*Work, error) {
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

	if len(works) < limit {
		slog.Debug("fetched fewer works than limit", "fetched", len(works), "limit", limit)
	}

	if len(works) > limit {
		works = works[:limit]
	}
	return works, nil
}

// ListArticlesByIDs implements [Client]
func (c *RestyClient) ListArticlesByIDs(ctx context.Context, ids []string) ([]*Work, error) {
	if len(ids) == 0 {
		return []*Work{}, nil
	}

	selectFields := getWorkFields()
	idChunks := utils.Chunks(ids, MaxIDsPerRequest)

	return fetchParallel(ctx, idChunks, func(ctx context.Context, idChunk []string) (*WorksResponse, error) {
		joinedIDs := strings.Join(idChunk, "|")
		queryParams := map[string]string{
			"select":   strings.Join(selectFields, ","),
			"filter":   fmt.Sprintf("ids.openalex:%s,type:types/article", joinedIDs),
			"per_page": fmt.Sprintf("%d", MaxIDsPerRequest),
		}
		return c.fetchWorks(ctx, queryParams)
	})
}

func fetchParallel[I any](ctx context.Context, inputs []I, fetch func(context.Context, I) (*WorksResponse, error)) ([]*Work, error) {
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

	var works []*Work
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
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			fields = append(fields, jsonTag)
		}
	}
	return fields
}

// WorkToArticle transforms a Work to an Article.
func WorkToArticle(work *Work) *models.Article {
	ids := collections.NewSet[string]()
	for source, id := range work.IDs {
		realID := id
		if source == "doi" {
			realID = extractDOI(id)
		}
		ids.Add(fmt.Sprintf("%s:%s", source, realID))
	}
	var authors []string
	for _, author := range work.Authorships {
		authors = append(authors, invertName(author.Author.DisplayName))
	}
	var journal *string
	if work.PrimaryLocation != nil && work.PrimaryLocation.Source != nil {
		journal = &work.PrimaryLocation.Source.DisplayName
	}
	var doi *string
	if work.DOI != nil {
		doiVal := extractDOI(*work.DOI)
		doi = &doiVal
	}
	var permalink *string
	if work.PrimaryLocation != nil && work.PrimaryLocation.LandingPageURL != nil {
		permalink = work.PrimaryLocation.LandingPageURL
	}
	references := make([]*models.Article, len(work.ReferencedWorks))
	for i, reference := range work.ReferencedWorks {
		references[i] = ReferenceToArticle(reference)
	}
	keywords := collections.NewSet[string]()
	for _, keyword := range work.Keywords {
		keywords.Add(keyword.DisplayName)
	}
	abstract := invertAbstract(work.AbstractInvertedIndex)
	return &models.Article{
		Label:      work.ID,
		IDs:        ids,
		Authors:    authors,
		Year:       work.PublicationYear,
		Title:      work.Title,
		Journal:    journal,
		Volume:     work.Biblio.Volume,
		Issue:      work.Biblio.Issue,
		Page:       work.Biblio.FirstPage,
		DOI:        doi,
		Permalink:  permalink,
		TimesCited: &work.CitedByCount,
		References: references,
		Keywords:   keywords,
		Abstract:   &abstract,
		Rich:       true,
	}
}

// ReferenceToArticle transforms an OpenAlex ID to an Article.
func ReferenceToArticle(reference string) *models.Article {
	return &models.Article{
		Label:     reference,
		IDs:       collections.NewSet(fmt.Sprintf("openalex:%s", reference)),
		Permalink: &reference,
		Rich:      false,
	}
}

func extractDOI(doi string) string {
	doi = strings.TrimSpace(doi)
	prefixes := []string{
		"https://doi.org/",
		"http://doi.org/",
		"https://dx.doi.org/",
		"http://dx.doi.org/",
		"doi:",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(doi), prefix) {
			return doi[len(prefix):]
		}
	}
	return doi
}

func invertName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, ",") {
		return name
	}
	parts := strings.Fields(name)
	if len(parts) < 2 {
		return name
	}
	lastName := parts[len(parts)-1]
	firstNames := parts[:len(parts)-1]
	return fmt.Sprintf("%s, %s", lastName, strings.Join(firstNames, " "))
}

func invertAbstract(abstractInvertedIndex *map[string][]int) string {
	if abstractInvertedIndex == nil {
		return ""
	}
	length := 0
	for _, indices := range *abstractInvertedIndex {
		maxIndex := 0
		for _, index := range indices {
			if index > maxIndex {
				maxIndex = index
			}
		}
		if maxIndex > length {
			length = maxIndex
		}
	}
	words := make([]string, length+1)
	for word, indices := range *abstractInvertedIndex {
		for _, index := range indices {
			words[index] = word
		}
	}
	return strings.TrimSpace(strings.Join(words, " "))
}
