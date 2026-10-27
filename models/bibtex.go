package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/nickng/bibtex"
)

var (
	nonKeyChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

	diacriticMap = map[rune]string{
		'á': "a", 'à': "a", 'â': "a", 'ä': "a", 'ã': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
		'Á': "A", 'À': "A", 'Â': "A", 'Ä': "A", 'Ã': "A", 'Å': "A", 'Ā': "A", 'Ă': "A", 'Ą': "A",
		'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ė': "e", 'ę': "e", 'ě': "e",
		'É': "E", 'È': "E", 'Ê': "E", 'Ë': "E", 'Ē': "E", 'Ĕ': "E", 'Ė': "E", 'Ę': "E", 'Ě': "E",
		'í': "i", 'ì': "i", 'î': "i", 'ï': "i", 'ī': "i", 'ĭ': "i", 'į': "i", 'ı': "i",
		'Í': "I", 'Ì': "I", 'Î': "I", 'Ï': "I", 'Ī': "I", 'Ĭ': "I", 'Į': "I", 'İ': "I",
		'ó': "o", 'ò': "o", 'ô': "o", 'ö': "o", 'õ': "o", 'ō': "o", 'ŏ': "o", 'ő': "o", 'ø': "o",
		'Ó': "O", 'Ò': "O", 'Ô': "O", 'Ö': "O", 'Õ': "O", 'Ō': "O", 'Ŏ': "O", 'Ő': "O", 'Ø': "O",
		'ú': "u", 'ù': "u", 'û': "u", 'ü': "u", 'ũ': "u", 'ū': "u", 'ŭ': "u", 'ů': "u", 'ű': "u", 'ų': "u",
		'Ú': "U", 'Ù': "U", 'Û': "U", 'Ü': "U", 'Ũ': "U", 'Ū': "U", 'Ŭ': "U", 'Ů': "U", 'Ű': "U", 'Ų': "U",
		'ý': "y", 'ÿ': "y", 'Ý': "Y", 'Ÿ': "Y",
		'ç': "c", 'ć': "c", 'ĉ': "c", 'ċ': "c", 'č': "c",
		'Ç': "C", 'Ć': "C", 'Ĉ': "C", 'Ċ': "C", 'Č': "C",
		'ď': "d", 'đ': "d", 'Ď': "D", 'Đ': "D", 'ð': "d", 'Ð': "D",
		'ĝ': "g", 'ğ': "g", 'ġ': "g", 'ģ': "g", 'Ĝ': "G", 'Ğ': "G", 'Ġ': "G", 'Ģ': "G",
		'ĥ': "h", 'ħ': "h", 'Ĥ': "H", 'Ħ': "H",
		'ĵ': "j", 'Ĵ': "J",
		'ķ': "k", 'Ķ': "K",
		'ĺ': "l", 'ļ': "l", 'ľ': "l", 'ŀ': "l", 'ł': "l",
		'Ĺ': "L", 'Ļ': "L", 'Ľ': "L", 'Ŀ': "L", 'Ł': "L",
		'ñ': "n", 'ń': "n", 'ņ': "n", 'ň': "n",
		'Ñ': "N", 'Ń': "N", 'Ņ': "N", 'Ň': "N",
		'ŕ': "r", 'ŗ': "r", 'ř': "r", 'Ŕ': "R", 'Ŗ': "R", 'Ř': "R",
		'ś': "s", 'ŝ': "s", 'ş': "s", 'š': "s",
		'Ś': "S", 'Ŝ': "S", 'Ş': "S", 'Š': "S",
		'ţ': "t", 'ť': "t", 'ŧ': "t", 'Ţ': "T", 'Ť': "T", 'Ŧ': "T",
		'ŵ': "w", 'Ŵ': "W",
		'ź': "z", 'ż': "z", 'ž': "z", 'Ź': "Z", 'Ż': "Z", 'Ž': "Z",
		'ß': "ss", 'æ': "ae", 'Æ': "AE", 'œ': "oe", 'Œ': "OE", 'þ': "th", 'Þ': "TH",
	}

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

// GenerateCitationKey generates a clean, URL-safe citation key for an article.
func GenerateCitationKey(a *Article) string {
	if a == nil {
		return "article"
	}

	// 1. Primary: SimpleId (first author + year, e.g. "vaswani2017")
	if simpleID := a.SimpleId(); simpleID != nil && *simpleID != "" {
		return sanitizeCiteKey(*simpleID)
	}

	// 2. Fallback: DOI
	if a.DOI != nil && *a.DOI != "" {
		cleanDOI := utils.ExtractDOI(*a.DOI)
		return sanitizeCiteKey("doi_" + cleanDOI)
	}

	// 3. Fallback: OpenAlex ID
	if id, ok := a.ID("openalex"); ok && id != "" {
		trimmed := strings.TrimPrefix(id, "https://openalex.org/")
		return sanitizeCiteKey(trimmed)
	}

	// 4. Fallback: Article Key
	if key := a.Key(); key != nil && *key != "" {
		return sanitizeCiteKey(*key)
	}

	// 5. Fallback: Article Label
	if a.Label != "" {
		return sanitizeCiteKey(a.Label)
	}

	return "article"
}

func cleanDiacritics(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if replacement, ok := diacriticMap[r]; ok {
			sb.WriteString(replacement)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func sanitizeCiteKey(s string) string {
	s = cleanDiacritics(s)
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

// ToBibEntry converts the Article into a [*bibtex.BibEntry].
// If citeKey is provided, it will be used as the citation key; otherwise, an automatic key is generated.
func (a *Article) ToBibEntry(citeKey ...string) *bibtex.BibEntry {
	if a == nil {
		return nil
	}

	entryType := "misc"
	if a.Journal != nil && *a.Journal != "" {
		entryType = "article"
	}

	key := GenerateCitationKey(a)
	if len(citeKey) > 0 && citeKey[0] != "" {
		key = citeKey[0]
	}

	entry := bibtex.NewBibEntry(entryType, key)

	if len(a.Authors) > 0 {
		entry.AddField("author", bibtex.NewBibConst(strings.Join(a.Authors, " and ")))
	}
	if a.Title != nil && *a.Title != "" {
		entry.AddField("title", bibtex.NewBibConst(*a.Title))
	}
	if a.Journal != nil && *a.Journal != "" {
		entry.AddField("journal", bibtex.NewBibConst(*a.Journal))
	}
	if a.Year != nil {
		entry.AddField("year", bibtex.NewBibConst(fmt.Sprintf("%d", *a.Year)))
	}
	if a.Volume != nil && *a.Volume != "" {
		entry.AddField("volume", bibtex.NewBibConst(*a.Volume))
	}
	if a.Issue != nil && *a.Issue != "" {
		entry.AddField("number", bibtex.NewBibConst(*a.Issue))
	}
	if a.Page != nil && *a.Page != "" {
		pages := strings.ReplaceAll(*a.Page, "-", "--")
		pages = strings.ReplaceAll(pages, "----", "--")
		entry.AddField("pages", bibtex.NewBibConst(pages))
	}
	if a.DOI != nil && *a.DOI != "" {
		entry.AddField("doi", bibtex.NewBibConst(utils.ExtractDOI(*a.DOI)))
	}
	if link := a.GetPermalink(); link != nil && *link != "" {
		entry.AddField("url", bibtex.NewBibConst(*link))
	}
	if a.Keywords != nil && a.Keywords.Len() > 0 {
		entry.AddField("keywords", bibtex.NewBibConst(strings.Join(a.Keywords.Items(), ", ")))
	}
	if a.Abstract != nil && *a.Abstract != "" {
		cleanAbstract := strings.Join(strings.Fields(*a.Abstract), " ")
		entry.AddField("abstract", bibtex.NewBibConst(cleanAbstract))
	}

	return entry
}

// ToBibTeX formats a single Article into a BibTeX entry string.
func (a *Article) ToBibTeX(citeKey ...string) string {
	if a == nil {
		return ""
	}
	entry := a.ToBibEntry(citeKey...)
	if entry == nil {
		return ""
	}
	return entry.PrettyString(bibtex.WithKeyOrder(defaultBibtexKeyOrder))
}

// ToBibTeX formats a collection of Articles into a concatenated BibTeX string,
// automatically disambiguating duplicate citation keys.
func (a Articles) ToBibTeX() string {
	if len(a) == 0 {
		return ""
	}

	bib := bibtex.NewBibTex()
	seenKeys := make(map[string]int)

	for _, art := range a {
		if art == nil {
			continue
		}
		baseKey := GenerateCitationKey(art)
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

		entry := art.ToBibEntry(finalKey)
		if entry != nil {
			bib.AddEntry(entry)
		}
	}

	return bib.PrettyString(bibtex.WithKeyOrder(defaultBibtexKeyOrder))
}
