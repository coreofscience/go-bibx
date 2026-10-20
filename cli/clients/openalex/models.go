package openalex

// AuthorPosition represents the position of an author in the authorship list.
type AuthorPosition string

const (
	// AuthorPositionFirst indicates the first author.
	AuthorPositionFirst AuthorPosition = "first"
	// AuthorPositionMiddle indicates a middle author.
	AuthorPositionMiddle AuthorPosition = "middle"
	// AuthorPositionLast indicates the last author.
	AuthorPositionLast AuthorPosition = "last"
)

// Author represents an author entity in OpenAlex.
type Author struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	ORCID       *string `json:"orcid"`
}

// WorkAuthorship links an [Author] to a work, including authorship order and corresponding status.
type WorkAuthorship struct {
	AuthorPosition  AuthorPosition `json:"author_position"`
	Author          Author         `json:"author"`
	IsCorresponding bool           `json:"is_corresponding"`
}

// WorkKeyword represents a keyword concept associated with a work in OpenAlex.
type WorkKeyword struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Score       float64 `json:"score"`
}

// WorkBiblio contains bibliographic publication details such as volume, issue, and pages.
type WorkBiblio struct {
	Volume    *string `json:"volume"`
	Issue     *string `json:"issue"`
	FirstPage *string `json:"first_page"`
	LastPage  *string `json:"last_page"`
}

// WorkLocationSource represents the venue or source of a work (e.g. journal or repository).
type WorkLocationSource struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

// WorkLocation details the access location, URLs, and source venue of a work.
type WorkLocation struct {
	IsOpenAccess   bool                `json:"is_oa"`
	LandingPageURL *string             `json:"landing_page_url"`
	PDFUrl         *string             `json:"pdf_url"`
	Source         *WorkLocationSource `json:"source"`
}

// Work represents a scholarly work entity returned by the OpenAlex API.
type Work struct {
	ID                    string            `json:"id"`
	IDs                   map[string]string `json:"ids"`
	DOI                   *string           `json:"doi"`
	Title                 *string           `json:"title"`
	PublicationYear       *int              `json:"publication_year"`
	Authorships           []WorkAuthorship  `json:"authorships"`
	CitedByCount          int               `json:"cited_by_count"`
	Keywords              []WorkKeyword     `json:"keywords"`
	AbstractInvertedIndex *map[string][]int `json:"abstract_inverted_index"`
	ReferencedWorks       []string          `json:"referenced_works"`
	Biblio                WorkBiblio        `json:"biblio"`
	PrimaryLocation       *WorkLocation     `json:"primary_location"`
}

// ResponseMeta contains pagination metadata returned by OpenAlex list responses.
type ResponseMeta struct {
	Count   int `json:"count"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// WorksResponse represents a paginated API response containing a list of works from OpenAlex.
type WorksResponse struct {
	Meta  ResponseMeta `json:"meta"`
	Works []*Work      `json:"results"`
}
