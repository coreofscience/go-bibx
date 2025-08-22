package clients

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strings"

	"golang.org/x/sync/errgroup"
	"resty.dev/v3"
)

const (
	MaxWorksPerPage       = 200
	MaxIdsPerRequest      = 80
	MaxConcurrentRequests = 5
)

type AuthorPosition string

const (
	AuthorPositionFirst  AuthorPosition = "first"
	AuthorPositionMiddle AuthorPosition = "middle"
	AuthorPositionLast   AuthorPosition = "last"
)

type Author struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	ORCID       *string `json:"orcid"`
}

type WorkAuthorship struct {
	AuthorPosition  AuthorPosition `json:"author_position"`
	Author          Author         `json:"author"`
	IsCorresponding bool           `json:"is_corresponding"`
}

type WorkKeyword struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Score       float64 `json:"score"`
}

type WorkBiblio struct {
	Volume    *string `json:"volume"`
	Issue     *string `json:"issue"`
	FirstPage *string `json:"first_page"`
	LastPage  *string `json:"last_page"`
}

type WorkLocationSource struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type WorkLocation struct {
	IsOpenAcces    bool                `json:"is_oa"`
	LandingPageUrl *string             `json:"landing_page_url"`
	PDFUrl         *string             `json:"pdf_url"`
	Source         *WorkLocationSource `json:"source"`
}

type Work struct {
	ID              string            `json:"id"`
	IDs             map[string]string `json:"ids"`
	DOI             *string           `json:"doi"`
	Title           *string           `json:"title"`
	PublicationYear *int              `json:"publication_year"`
	Authorships     []WorkAuthorship  `json:"authorships"`
	CitedByCount    int               `json:"cited_by_count"`
	Keywords        []WorkKeyword     `json:"keywords"`
	ReferencedWorks []string          `json:"referenced_works"`
	Biblio          WorkBiblio        `json:"biblio"`
	PrimaryLocation *WorkLocation     `json:"primary_location"`
}

type ResponseMeta struct {
	Count   int `json:"count"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type WorksResponse struct {
	Meta  ResponseMeta `json:"meta"`
	Works []Work       `json:"results"`
}

type ListRecentArticlesParams struct {
	Query string
	Limit *int
}

type OpenAlexClient interface {
	ListRecentArticles(ctx context.Context, params *ListRecentArticlesParams) ([]Work, error)
}

type openAlexClient struct {
	baseURL     string
	client      *resty.Client
	baseHeaders map[string]string
}

type NewOpenAlexClientParams struct {
	BaseURL     string
	HTTPClient  *resty.Client
	Email       string
	BaseHeaders map[string]string
}

func NewOpenAlexClient(params *NewOpenAlexClientParams) OpenAlexClient {
	if params.BaseURL == "" {
		params.BaseURL = "https://api.openalex.org"
	}
	if params.HTTPClient == nil {
		params.HTTPClient = resty.New()
	}
	if params.Email != "" {
		params.Email = "technology@coreofscience.org"
	}
	if params.BaseHeaders == nil {
		params.BaseHeaders = map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
			"User-Agent":   fmt.Sprintf("Go/resty/go-bibx mailto:%s", params.Email),
		}
	}
	return &openAlexClient{
		baseURL:     params.BaseURL,
		client:      params.HTTPClient,
		baseHeaders: params.BaseHeaders,
	}
}

func (c *openAlexClient) ListRecentArticles(ctx context.Context, params *ListRecentArticlesParams) ([]Work, error) {
	if params == nil {
		return nil, errors.New("params cannot be nil")
	}

	slelectFields := jsonFields(Work{})
	queryFilter := fmt.Sprintf(
		"title_and_abstract.search:%s",
		strings.ReplaceAll(params.Query, " ", "+"),
	)
	filterParts := []string{
		queryFilter,
		"type:types/article",
		"cited_by_count:>1",
	}

	group, ctx := errgroup.WithContext(ctx)
	pages := make(chan int)
	responses := make(chan *WorksResponse)

	if params.Limit == nil || *params.Limit <= 0 {
		defaultLimit := 600
		params.Limit = &defaultLimit
	}

	maxPages := int(math.Ceil(float64(*params.Limit) / MaxWorksPerPage))
	concurrency := min(MaxConcurrentRequests, maxPages)
	log.Printf("Fetching up to %d pages with concurrency %d", maxPages, concurrency)

	for i := range concurrency {
		goroutineIndex := i
		group.Go(func() error {
			for page := range pages {
				log.Printf("Fetching page %d from goroutine %d", page, goroutineIndex)
				queryParams := map[string]string{
					"select":   strings.Join(slelectFields, ","),
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
					log.Printf("Error response: %s", response.String())
					return fmt.Errorf("error fetching works: %s", response.Status())
				}
				result := response.Result().(*WorksResponse)
				if result == nil {
					return errors.New("error parsing works response")
				}
				if len(result.Works) == 0 {
					return nil
				}
				responses <- result
			}
			return nil
		})
	}

	for page := 0; page < int(maxPages); page++ {
		pages <- page + 1
	}
	close(pages)

	works := make([]Work, 0)
	go func() {
		for res := range responses {
			if res != nil {
				works = append(works, res.Works...)
			}
		}
	}()

	err := group.Wait()
	if err != nil {
		return nil, fmt.Errorf("error fetching works in parallel: %w", err)
	}
	close(responses)

	return works, nil
}

// List all the JSON fields for an arbitrary struct
func jsonFields(t any) []string {
	var fields []string
	v := reflect.ValueOf(t)
	if v.Kind() == reflect.Ptr {
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
