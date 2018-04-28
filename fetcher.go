package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var errRedirectHostMismatch = errors.New("http redirect host mismath")
var errRequestNotOK = errors.New("status code is not 'OK'")

type fetcher struct {
	client   *http.Client
	throttle <-chan time.Time

	inCh  <-chan *page
	outCh chan<- *page
	errCh chan<- *page
}

func redirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) > 3 {
		return http.ErrUseLastResponse
	}

	if req.URL.Host != via[0].URL.Host {
		return errRedirectHostMismatch
	}

	return nil
}

func newFetcher(timeout time.Duration, requestsPerMinuteLimit uint, inCh <-chan *page, outCh chan<- *page, errCh chan<- *page) *fetcher {
	f := &fetcher{
		client: &http.Client{Timeout: timeout, CheckRedirect: redirectPolicy},
		inCh:   inCh,
		outCh:  outCh,
		errCh:  errCh,
	}

	if requestsPerMinuteLimit > 0 {
		f.throttle = time.Tick(time.Minute / time.Duration(requestsPerMinuteLimit))
	}

	return f
}

func (f *fetcher) Run() {
	for page := range f.inCh {
		log.Debugw("recv", "url", page.url)

		body, err := f.fetch(page.url)

		if err != nil {
			log.Debugw("err", "url", page.url)
			f.errCh <- page
		} else {
			page.body = body
			log.Debugw("send", "url", page.url)
			f.outCh <- page
		}
	}
}

func (f *fetcher) fetch(u *url.URL) ([]byte, error) {
	if f.throttle != nil {
		<-f.throttle
	}

	resp, err := f.client.Get(u.String())
	if err != nil {
		log.Errorw("request", "url", u, "err", err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Errorw("request", "url", u, "status_code", resp.StatusCode)
		return nil, errRequestNotOK
	}

	return ioutil.ReadAll(resp.Body)
}
