package main

import (
	"bytes"
	"net/url"
	"path/filepath"

	"github.com/PuerkitoBio/purell"
	"golang.org/x/net/html"
)

const purellFlags = (purell.FlagLowercaseHost |
	purell.FlagLowercaseScheme |
	purell.FlagSortQuery |
	purell.FlagRemoveDotSegments |
	purell.FlagRemoveDuplicateSlashes |
	purell.FlagRemoveEmptyQuerySeparator |
	purell.FlagRemoveFragment |
	purell.FlagRemoveTrailingSlash)

var (
	linkHTMLTags = map[string][]string{
		"a":      []string{"href"},
		"head":   []string{"profile"},
		"iframe": []string{"longdesc", "src"},
		"q":      []string{"cite"},
	}

	validExtension = map[string]bool{
		"":      true,
		".html": true,
	}
)

type parser struct {
	seed  *url.URL
	inCh  <-chan *page
	outCh chan<- *page
}

func newParser(seed *url.URL, inCh <-chan *page, outCh chan<- *page) *parser {
	return &parser{
		seed:  seed,
		inCh:  inCh,
		outCh: outCh,
	}
}

func (p *parser) Run() {
	for page := range p.inCh {
		log.Debugw("recv", "url", page.url)

		page.children = p.parse(page.url, page.body)

		log.Debugw("send", "url", page.url, "children", len(page.children))

		p.outCh <- page
	}
}

func (p *parser) parse(parent *url.URL, body []byte) []*url.URL {
	urls := p.extractURLs(body)
	normalized := p.normalize(parent, urls)
	filtered := p.filter(normalized)

	log.Debugw("parse", "URLs", ToStrings(urls), "normalized", normalized, "filtered", ToStrings(filtered))

	return filtered
}

func (p *parser) extractURLs(body []byte) []*url.URL {
	result := make([]*url.URL, 0)
	tokenizer := html.NewTokenizer(bytes.NewReader(body))

	for {
		token := tokenizer.Next()

		switch {
		case token == html.ErrorToken:
			return result

		case token == html.StartTagToken:
			t := tokenizer.Token()

			for _, attr := range extractAttrs(&t, linkHTMLTags) {
				u, err := url.Parse(attr)
				if err != nil {
					continue
				}

				result = append(result, u)
			}
		}
	}
}

func (p *parser) normalize(parent *url.URL, urls []*url.URL) []string {
	result := make([]string, 0)
	for _, u := range urls {
		if u == nil {
			continue
		}

		normalized := purell.NormalizeURL(parent.ResolveReference(u), purellFlags)
		result = append(result, normalized)
	}

	return result
}

func (p *parser) filter(urlStrings []string) []*url.URL {
	result := make([]*url.URL, 0)
	seen := map[string]bool{p.seed.String(): true}

	for _, str := range urlStrings {
		if seen[str] {
			continue
		}

		u, err := url.Parse(str)
		if !(err == nil && p.isValid(u)) {
			continue
		}

		result = append(result, u)
		seen[str] = true
	}

	return result
}

func (p *parser) isValid(u *url.URL) bool {
	// To keep it simple, only allow exact domain matches (i.e. no subdomains)
	return u.Hostname() == p.seed.Hostname() &&
		(u.Scheme == "http" || u.Scheme == "https") &&
		(u.Path == "" || validExtension[filepath.Ext(u.Path)])
}

func extractAttrs(token *html.Token, tags map[string][]string) []string {
	var attrs = make([]string, 0)

	if targetAttrs, ok := tags[token.Data]; ok {
		for _, attr := range targetAttrs {
			if attrVal := getAttr(token, attr); attrVal != "" {
				attrs = append(attrs, attrVal)
				break
			}
		}
	}

	return attrs
}

func getAttr(t *html.Token, key string) string {
	var attr string

	for _, a := range t.Attr {
		if a.Key == key {
			attr = a.Val
			break
		}
	}

	return attr
}
