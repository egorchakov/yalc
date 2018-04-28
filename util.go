package main

import (
	"net/url"
)

func MapURLs(vs []*url.URL, f func(*url.URL) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func ToStrings(urls []*url.URL) []string {
	return MapURLs(urls, (*url.URL).String)
}
