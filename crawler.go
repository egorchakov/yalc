package main

import (
	"net/url"
	"runtime"
	"time"
)

const (
	chanSize = 1 << 10
)

var (
	// Fetchers are IO-bound
	// This number also limits the number of concurrent connections
	numFetchers = runtime.NumCPU() * 32

	// Parsers are CPU-bound
	numParsers = runtime.NumCPU()
)

type Crawler interface {
	Crawl(seed *url.URL, timeout time.Duration, requestsPerMinuteLimit uint) Result
}

type crawler struct{}

func NewCrawler() Crawler {
	return &crawler{}
}

func (c *crawler) Crawl(seed *url.URL, timeout time.Duration, requestsPerMinuteLimit uint) Result {
	managerToFetcherCh := make(chan *page, chanSize)
	fetcherToParserCh := make(chan *page, chanSize)
	parserToManagerCh := make(chan *page, chanSize)
	errorCh := make(chan *page, chanSize)
	resultCh := make(chan Result)

	manager := newManager(seed, parserToManagerCh, managerToFetcherCh, errorCh, resultCh)
	fetcher := newFetcher(timeout, requestsPerMinuteLimit, managerToFetcherCh, fetcherToParserCh, errorCh)
	parser := newParser(seed, fetcherToParserCh, parserToManagerCh)

	log.Infow("start", "seed", seed, "timeout", timeout, "rate limit (req/min)", requestsPerMinuteLimit)

	for i := 0; i < numFetchers; i++ {
		go fetcher.Run()
	}

	for i := 0; i < numParsers; i++ {
		go parser.Run()
	}

	go manager.Run()

	return <-resultCh
}
