package main

import (
	"net/url"
	"reflect"
	"testing"

	"go.uber.org/zap"
)

func init() {
	logger, _ := zap.NewDevelopment()
	log = logger.Sugar()
}

func TestExtractURLs(t *testing.T) {
	p := &parser{}

	var body = []byte(`
	<head profile="http://head.profile">
	<a href="http://a.href"> </a>
	<iframe src="http://iframe.src"> </iframe>
	<q cite="http://q.cite"></q>

	<link href="http://link.href">
	<img src="http://img.src">
	<script src="http://script.src">
	`)

	expectedURLs := []string{
		"http://head.profile",
		"http://a.href",
		"http://iframe.src",
		"http://q.cite",
	}

	urls := p.extractURLs(body)

	if !reflect.DeepEqual(ToStrings(urls), expectedURLs) {
		t.Logf("expected: %+v, got: %+v", expectedURLs, urls)
		t.Fail()
	}
}

func TestFilterURLs(t *testing.T) {
	seed, _ := url.Parse("https://example.com")
	p := &parser{seed: seed}

	inputURLs := []string{
		"https://example.com",
		"https://example.com/path",
		"https://example.com/path",
		"https://example.com/page.html",
		"https://example.com/file.pdf",
		"sftp://example.com/server",
		"https://not.example.com/path",
		"https://notexample.com/path",
	}

	expectedURLs := []string{
		"https://example.com/path",
		"https://example.com/page.html",
	}

	urls := p.filter(inputURLs)

	if !reflect.DeepEqual(ToStrings(urls), expectedURLs) {
		t.Logf("expected: %+v, got: %+v", expectedURLs, urls)
		t.Fail()
	}
}
