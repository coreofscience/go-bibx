package openalex

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
	IsOpenAccess   bool                `json:"is_oa"`
	LandingPageUrl *string             `json:"landing_page_url"`
	PDFUrl         *string             `json:"pdf_url"`
	Source         *WorkLocationSource `json:"source"`
}

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

type ResponseMeta struct {
	Count   int `json:"count"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type WorksResponse struct {
	Meta  ResponseMeta `json:"meta"`
	Works []Work       `json:"results"`
}
