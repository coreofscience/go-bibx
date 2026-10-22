package search

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"
)

// Document represents an article document for BM25 search.
type Document struct {
	ID       string
	Title    string
	Keywords []string
	Abstract string
	Authors  []string
}

// Result represents a scored document from a BM25 search.
type Result struct {
	ID    string
	Score float64
}

// FieldWeights defines the weights applied to each document field in BM25.
type FieldWeights struct {
	Title    float64
	Keywords float64
	Abstract float64
	Authors  float64
}

// Options configures the BM25 search algorithm parameters.
type Options struct {
	K1          float64
	B           float64
	PhraseBoost float64
	Weights     FieldWeights
}

// DefaultOptions returns the standard BM25 options with field weighting and phrase boost.
func DefaultOptions() Options {
	return Options{
		K1:          1.2,
		B:           0.75,
		PhraseBoost: 2.0,
		Weights: FieldWeights{
			Title:    3.0,
			Keywords: 2.0,
			Abstract: 1.0,
			Authors:  2.0,
		},
	}
}

// Option modifies search options.
type Option func(*Options)

// WithK1 sets the k1 saturation parameter.
func WithK1(k1 float64) Option {
	return func(o *Options) {
		o.K1 = k1
	}
}

// WithB sets the b document length normalization parameter.
func WithB(b float64) Option {
	return func(o *Options) {
		o.B = b
	}
}

// WithPhraseBoost sets the bonus added for title phrase matches.
func WithPhraseBoost(boost float64) Option {
	return func(o *Options) {
		o.PhraseBoost = boost
	}
}

// WithFieldWeights sets the field weights.
func WithFieldWeights(weights FieldWeights) Option {
	return func(o *Options) {
		o.Weights = weights
	}
}

// Tokenize normalizes text by lowercasing, stripping punctuation, and splitting on whitespace.
func Tokenize(text string) []string {
	var sb strings.Builder
	for _, r := range text {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(unicode.ToLower(r))
		}
	}
	return strings.Fields(sb.String())
}

type tokenizedDoc struct {
	doc            Document
	length         int
	titleCounts    map[string]int
	keywordsCounts map[string]int
	abstractCounts map[string]int
	authorsCounts  map[string]int
}

func countTokens(tokens []string) map[string]int {
	counts := make(map[string]int, len(tokens))
	for _, t := range tokens {
		counts[t]++
	}
	return counts
}

// BM25Search searches the given documents using the BM25 algorithm with field weighting and phrase boost.
func BM25Search(docs []Document, query string, limit int, opts ...Option) []Result {
	if len(docs) == 0 || limit <= 0 {
		return nil
	}

	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return nil
	}

	queryTokens := Tokenize(trimmedQuery)
	if len(queryTokens) == 0 {
		return nil
	}

	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Deduplicate query terms while preserving order
	uniqueTerms := make([]string, 0, len(queryTokens))
	seenTerms := make(map[string]bool, len(queryTokens))
	for _, t := range queryTokens {
		if !seenTerms[t] {
			seenTerms[t] = true
			uniqueTerms = append(uniqueTerms, t)
		}
	}

	n := len(docs)
	tokenizedDocs := make([]tokenizedDoc, n)
	totalLength := 0

	for i, doc := range docs {
		titleTokens := Tokenize(doc.Title)

		var keywordsTokens []string
		for _, kw := range doc.Keywords {
			keywordsTokens = append(keywordsTokens, Tokenize(kw)...)
		}

		abstractTokens := Tokenize(doc.Abstract)

		var authorsTokens []string
		for _, author := range doc.Authors {
			authorsTokens = append(authorsTokens, Tokenize(author)...)
		}

		docLen := len(titleTokens) + len(keywordsTokens) + len(abstractTokens) + len(authorsTokens)
		totalLength += docLen

		tokenizedDocs[i] = tokenizedDoc{
			doc:            doc,
			length:         docLen,
			titleCounts:    countTokens(titleTokens),
			keywordsCounts: countTokens(keywordsTokens),
			abstractCounts: countTokens(abstractTokens),
			authorsCounts:  countTokens(authorsTokens),
		}
	}

	avgdl := float64(totalLength) / float64(n)

	// Calculate document frequency n(q) for each unique query term
	docFreq := make(map[string]int, len(uniqueTerms))
	for _, term := range uniqueTerms {
		count := 0
		for _, td := range tokenizedDocs {
			fTitle := td.titleCounts[term]
			fKeywords := td.keywordsCounts[term]
			fAbstract := td.abstractCounts[term]
			fAuthors := td.authorsCounts[term]
			if fTitle > 0 || fKeywords > 0 || fAbstract > 0 || fAuthors > 0 {
				count++
			}
		}
		docFreq[term] = count
	}

	// Compute Robertson-Spärck Jones / Lucene IDF: ln(1 + (N - n(q) + 0.5) / (n(q) + 0.5))
	idf := make(map[string]float64, len(uniqueTerms))
	for _, term := range uniqueTerms {
		nq := docFreq[term]
		idf[term] = math.Log(1.0 + (float64(n)-float64(nq)+0.5)/(float64(nq)+0.5))
	}

	queryLower := strings.ToLower(trimmedQuery)

	results := make([]Result, 0, n)
	for _, td := range tokenizedDocs {
		termScore := 0.0
		hasMatch := false

		docLenRatio := 0.0
		if avgdl > 0 {
			docLenRatio = float64(td.length) / avgdl
		}
		lenNorm := options.K1 * (1.0 - options.B + options.B*docLenRatio)

		for _, term := range uniqueTerms {
			fTitle := float64(td.titleCounts[term])
			fKeywords := float64(td.keywordsCounts[term])
			fAbstract := float64(td.abstractCounts[term])
			fAuthors := float64(td.authorsCounts[term])

			f := options.Weights.Title*fTitle +
				options.Weights.Keywords*fKeywords +
				options.Weights.Abstract*fAbstract +
				options.Weights.Authors*fAuthors

			if f <= 0 {
				continue
			}

			hasMatch = true
			denom := f + lenNorm
			if denom > 0 {
				termScore += idf[term] * (f * (options.K1 + 1.0)) / denom
			}
		}

		if !hasMatch || termScore <= 0 {
			continue
		}

		finalScore := termScore
		if strings.Contains(strings.ToLower(td.doc.Title), queryLower) {
			finalScore += options.PhraseBoost
		}

		results = append(results, Result{
			ID:    td.doc.ID,
			Score: finalScore,
		})
	}

	slices.SortFunc(results, func(a, b Result) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})

	if limit > len(results) {
		limit = len(results)
	}

	return results[:limit]
}
