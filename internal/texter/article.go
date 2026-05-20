package texter

import (
	"strings"
	"unicode"

	"github.com/coreofscience/go-bibx/models"
)

type ArticleTexter interface {
	ExtractText(a *models.Article) string
	CleanText(s string) string
}

type DefaultArticleTexter struct{}

func NewDefaultArticleTexter() *DefaultArticleTexter {
	return &DefaultArticleTexter{}
}

// ExtractText implements the [ArticleTexter] interface.
func (t *DefaultArticleTexter) ExtractText(a *models.Article) string {
	parts := make([]string, 0, 4)
	if a.Title != nil {
		parts = append(parts, *a.Title)
	}
	if a.Abstract != nil {
		parts = append(parts, *a.Abstract)
	}
	if a.Keywords.Len() > 0 {
		parts = append(parts, a.Keywords.Items()...)
	}
	rawString := strings.Join(parts, " ")
	return t.CleanText(rawString)
}

// CleanText implements the [ArticleTexter] interface.
func (t *DefaultArticleTexter) CleanText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, s)
}
