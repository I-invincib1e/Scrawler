package parse

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Signals struct {
	URL        string   `json:"url"`
	Title      string   `json:"title"`
	MetaDesc   string   `json:"meta_description,omitempty"`
	Headings   []string `json:"headings"`
	Paragraphs []string `json:"paragraphs"`
	Links      []string `json:"links"`
}

func collapseWhitespace(s string) string {
	if s == "" {
		return ""
	}
	return strings.Join(strings.Fields(s), " ")
}

// ExtractSignals extracts minimal signals from a document and normalizes links.
func ExtractSignals(doc *goquery.Document, pageURL string) Signals {
	result := Signals{URL: pageURL}
	base, _ := url.Parse(pageURL)

	seenParagraph := make(map[string]struct{})
	seenLink := make(map[string]struct{})

	result.Title = collapseWhitespace(doc.Find("title").First().Text())
	if v, ok := doc.Find("meta[name='description']").Attr("content"); ok {
		result.MetaDesc = collapseWhitespace(v)
	}
	if result.MetaDesc == "" {
		if v, ok := doc.Find("meta[name='Description']").Attr("content"); ok {
			result.MetaDesc = collapseWhitespace(v)
		}
	}

	doc.Find("h1, h2, h3, h4, h5, h6").Each(func(_ int, s *goquery.Selection) {
		t := collapseWhitespace(s.Text())
		if t != "" {
			result.Headings = append(result.Headings, t)
		}
	})

	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		t := collapseWhitespace(s.Text())
		if len(t) <= 20 {
			return
		}
		if _, ok := seenParagraph[t]; ok {
			return
		}
		seenParagraph[t] = struct{}{}
		result.Paragraphs = append(result.Paragraphs, t)
	})

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || strings.TrimSpace(href) == "" {
			return
		}
		u, err := base.Parse(href)
		if err != nil {
			return
		}
		if u.Scheme == "" {
			u.Scheme = base.Scheme
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return
		}
		u.Fragment = ""
		abs := u.String()
		if _, ok := seenLink[abs]; ok {
			return
		}
		seenLink[abs] = struct{}{}
		result.Links = append(result.Links, abs)
	})

	return result
}
