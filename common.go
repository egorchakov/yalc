package main

import (
	"net/url"

	"go.uber.org/zap"
)

var log *zap.SugaredLogger

type page struct {
	url      *url.URL
	body     []byte
	children []*url.URL
}

// Result contains the output of the crawler
type Result struct {
	links      map[string][]string
	errorCount uint
}
