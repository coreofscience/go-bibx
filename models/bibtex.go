package models

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/nickng/bibtex"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	nonKeyChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

	defaultBibtexKeyOrder = []string{
		"author",
		"title",
		"journal",
		"year",
		"volume",
		"number",
		"pages",
		"doi",
		"url",
		"keywords",
		"abstract",
	}
)

// CitationKeyInfo provides metadata for an exported citation.
type CitationKeyInfo struct {
	ID          string `json:"id" jsonschema:"Work identifier in the collection"`
	CiteKey     string `json:"citekey" jsonschema:"Generated BibTeX citation key (disambiguated within the export)"`
	Title       string `json:"title,omitempty" jsonschema:"Work title"`
	Year        *int   `json:"year,omitempty" jsonschema:"Publication year"`
	FirstAuthor string `json:"first_author,omitempty" jsonschema:"Primary author name"`
}

// BibtexOption specifies configuration options for BibTeX generation.
type BibtexOption func(*bibtexOptions)

type bibtexOptions struct {
	omitAbstracts bool
	citeKey       string
}

// WithOmitAbstracts configures whether the abstract field should be omitted from BibTeX entries.
func WithOmitAbstracts(omit bool) BibtexOption {
	return func(o *bibtexOptions) {
		o.omitAbstracts = omit
	}
}

// WithCiteKey specifies a custom citation key for a single Work BibTeX entry.
func WithCiteKey(key string) BibtexOption {
	return func(o *bibtexOptions) {
		o.citeKey = key
	}
}

// SanitizeLaTeX escapes common LaTeX reserved characters (%, &, _, #) in plain text,
// avoiding double-escaping if a character is already preceded by a backslash.
func SanitizeLaTeX(s string) string {
	var sb strings.Builder
	runesList := []rune(s)
	for i, r := range runesList {
		switch r {
		case '%', '&', '_', '#':
			escaped := false
			if i > 0 && runesList[i-1] == '\\' {
				numBackslashes := 0
				for j := i - 1; j >= 0 && runesList[j] == '\\'; j-- {
					numBackslashes++
				}
				if numBackslashes%2 == 1 {
					escaped = true
				}
			}
			if !escaped {
				sb.WriteRune('\\')
			}
			sb.WriteRune(r)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// GenerateCitationKey generates a clean, URL-safe citation key for a work.
func GenerateCitationKey(w *Work) string {
	if w == nil {
		return "article"
	}

	// 1. Primary: SimpleId (first author + year, e.g. "vaswani2017")
	if simpleID := w.SimpleId(); simpleID != nil && *simpleID != "" {
		return sanitizeCiteKey(*simpleID)
	}

	// 2. Fallback: DOI
	if w.DOI != nil && *w.DOI != "" {
		cleanDOI := utils.ExtractDOI(*w.DOI)
		return sanitizeCiteKey("doi_" + cleanDOI)
	}

	// 3. Fallback: OpenAlex ID
	if id, ok := w.ID("openalex"); ok && id != "" {
		trimmed := strings.TrimPrefix(id, "https://openalex.org/")
		return sanitizeCiteKey(trimmed)
	}

	// 4. Fallback: Work Key
	if key := w.Key(); key != nil && *key != "" {
		return sanitizeCiteKey(*key)
	}

	// 5. Fallback: Work Label
	if w.Label != "" {
		return sanitizeCiteKey(w.Label)
	}

	return "article"
}

func cleanDiacritics(s string) (string, error) {
	t := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)),
		norm.NFC,
	)
	result, _, err := transform.String(t, s)
	if err != nil {
		return "", fmt.Errorf("error cleaning diacritics: %w", err)
	}
	return result, nil
}

func sanitizeCiteKey(s string) string {
	s, err := cleanDiacritics(s)
	if err != nil {
		return "article"
	}
	s = nonKeyChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	s = strings.ToLower(s)
	if s == "" {
		return "article"
	}
	return s
}

func keySuffix(n int) string {
	if n <= 0 {
		return ""
	}
	var buf []byte
	for n > 0 {
		n--
		buf = append([]byte{byte('a' + (n % 26))}, buf...)
		n /= 26
	}
	return string(buf)
}

// ToBibEntry converts the Work into a [*bibtex.BibEntry].
// Accepts optional [BibtexOption] modifiers (e.g. [WithCiteKey], [WithOmitAbstracts]).
func (w *Work) ToBibEntry(opts ...BibtexOption) *bibtex.BibEntry {
	if w == nil {
		return nil
	}

	options := bibtexOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	entryType := "misc"
	if w.Venue != nil && *w.Venue != "" {
		entryType = "article"
	}

	key := GenerateCitationKey(w)
	if options.citeKey != "" {
		key = options.citeKey
	}

	entry := bibtex.NewBibEntry(entryType, key)

	if len(w.Authors) > 0 {
		entry.AddField("author", bibtex.NewBibConst(strings.Join(w.Authors, " and ")))
	}
	if w.Title != nil && *w.Title != "" {
		entry.AddField("title", bibtex.NewBibConst(SanitizeLaTeX(*w.Title)))
	}
	if w.Venue != nil && *w.Venue != "" {
		entry.AddField("journal", bibtex.NewBibConst(SanitizeLaTeX(*w.Venue)))
	}
	if w.Year != nil {
		entry.AddField("year", bibtex.NewBibConst(fmt.Sprintf("%d", *w.Year)))
	}
	if w.Volume != nil && *w.Volume != "" {
		entry.AddField("volume", bibtex.NewBibConst(*w.Volume))
	}
	if w.Issue != nil && *w.Issue != "" {
		entry.AddField("number", bibtex.NewBibConst(*w.Issue))
	}
	if w.Page != nil && *w.Page != "" {
		pages := strings.ReplaceAll(*w.Page, "-", "--")
		pages = strings.ReplaceAll(pages, "----", "--")
		entry.AddField("pages", bibtex.NewBibConst(pages))
	}
	if w.DOI != nil && *w.DOI != "" {
		entry.AddField("doi", bibtex.NewBibConst(utils.ExtractDOI(*w.DOI)))
	}
	if link := w.GetPermalink(); link != nil && *link != "" {
		entry.AddField("url", bibtex.NewBibConst(*link))
	}
	if w.Keywords != nil && w.Keywords.Len() > 0 {
		items := w.Keywords.Items()
		sanitizedKeywords := make([]string, len(items))
		for i, kw := range items {
			sanitizedKeywords[i] = SanitizeLaTeX(kw)
		}
		entry.AddField("keywords", bibtex.NewBibConst(strings.Join(sanitizedKeywords, ", ")))
	}
	if !options.omitAbstracts && w.Abstract != nil && *w.Abstract != "" {
		cleanAbstract := strings.Join(strings.Fields(*w.Abstract), " ")
		entry.AddField("abstract", bibtex.NewBibConst(SanitizeLaTeX(cleanAbstract)))
	}

	return entry
}

// formatBibEntry formats a bibtex.BibEntry into standard BibTeX with {...} value delimiters,
// preserving LaTeX escapes without backslash-doubling.
func formatBibEntry(buf *bytes.Buffer, entry *bibtex.BibEntry, keyOrder []string) {
	if entry == nil {
		return
	}

	fmt.Fprintf(buf, "@%s{%s,\n", entry.Type, entry.CiteName)

	priority := make(map[string]int, len(keyOrder))
	for i, k := range keyOrder {
		priority[k] = i - len(keyOrder)
	}

	keys := make([]string, 0, len(entry.Fields))
	for k := range entry.Fields {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		pi, pj := priority[keys[i]], priority[keys[j]]
		return pi < pj || (pi == pj && keys[i] < keys[j])
	})

	tw := tabwriter.NewWriter(buf, 1, 4, 1, ' ', 0)
	for _, k := range keys {
		val := entry.Fields[k].String()
		if _, err := strconv.Atoi(val); err == nil && (k == "year" || k == "volume" || k == "number") {
			_, _ = fmt.Fprintf(tw, "    %s\t=\t%s,\n", k, val)
		} else {
			_, _ = fmt.Fprintf(tw, "    %s\t=\t{%s},\n", k, val)
		}
	}
	_ = tw.Flush()
	buf.WriteString("}\n")
}

// ToBibTeX formats a single Work into a BibTeX entry string.
func (w *Work) ToBibTeX(opts ...BibtexOption) string {
	if w == nil {
		return ""
	}
	entry := w.ToBibEntry(opts...)
	if entry == nil {
		return ""
	}
	var buf bytes.Buffer
	formatBibEntry(&buf, entry, defaultBibtexKeyOrder)
	return buf.String()
}

// ToBibTeXWithKeys formats a collection of Works into a concatenated BibTeX string,
// automatically disambiguating duplicate citation keys, and returns the slice of [CitationKeyInfo].
func (w Works) ToBibTeXWithKeys(opts ...BibtexOption) (string, []CitationKeyInfo) {
	if len(w) == 0 {
		return "", nil
	}

	options := bibtexOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	var buf bytes.Buffer
	seenKeys := make(map[string]int)
	keysInfo := make([]CitationKeyInfo, 0, len(w))

	for _, work := range w {
		if work == nil {
			continue
		}
		baseKey := GenerateCitationKey(work)
		count := seenKeys[baseKey]
		seenKeys[baseKey]++

		var finalKey string
		if count == 0 {
			finalKey = baseKey
		} else {
			suffix := keySuffix(count)
			finalKey = fmt.Sprintf("%s%s", baseKey, suffix)
			for seenKeys[finalKey] > 0 {
				count++
				suffix = keySuffix(count)
				finalKey = fmt.Sprintf("%s%s", baseKey, suffix)
			}
			seenKeys[finalKey]++
		}

		entry := work.ToBibEntry(WithCiteKey(finalKey), WithOmitAbstracts(options.omitAbstracts))
		if entry != nil {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			formatBibEntry(&buf, entry, defaultBibtexKeyOrder)

			var firstAuthor string
			if len(work.Authors) > 0 {
				firstAuthor = work.Authors[0]
			}
			var title string
			if work.Title != nil {
				title = *work.Title
			}
			id := work.Label
			if key := work.Key(); key != nil && *key != "" {
				id = *key
			}

			keysInfo = append(keysInfo, CitationKeyInfo{
				ID:          id,
				CiteKey:     finalKey,
				Title:       title,
				Year:        work.Year,
				FirstAuthor: firstAuthor,
			})
		}
	}

	return buf.String(), keysInfo
}

// ToBibTeX formats a collection of Works into a concatenated BibTeX string,
// automatically disambiguating duplicate citation keys.
func (w Works) ToBibTeX(opts ...BibtexOption) string {
	bibStr, _ := w.ToBibTeXWithKeys(opts...)
	return bibStr
}
