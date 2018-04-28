# yalc

yet another lil crawler

## Running
```
go build

./crawler https://example.com

cat example.com_sitemap.json | jq
```

## Implementation notes

The crawler is implemented as a cyclical "pipeline" with stages that communicate over channels. The stage types are:

1. Fetcher: performs GET requests on URLs. 
2. Parser: takes a page body produced by a fetcher and outputs relevant URLs from that page. 
3. Manager: tracks URLs and maintains a job queue. 

There are multiple fetchers and parsers, and a single manager.
