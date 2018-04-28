package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"go.uber.org/zap"
	kp "gopkg.in/alecthomas/kingpin.v2"
)

var app = kp.New("crawler", "single-host crawler")

var conf struct {
	Seed      *url.URL
	Timeout   time.Duration
	RateLimit uint

	Debug bool
	Trace bool
}

func parseArgs() {
	app.Arg("seed", "Seed URL").Required().URLVar(&conf.Seed)
	app.Flag("timeout", "GET request timeout").Default("10s").DurationVar(&conf.Timeout)
	app.Flag("rate", "Rate limit (in requests per minute)").Default("0").UintVar(&conf.RateLimit)
	app.Flag("debug", "Enable debug logs").BoolVar(&conf.Debug)

	kp.MustParse(app.Parse(os.Args[1:]))

	if conf.Seed.Host == "" {
		conf.Seed.Host = conf.Seed.Path
		conf.Seed.Path = ""
	}
}

func setupLogging() {
	logCfg := zap.NewDevelopmentConfig()
	logLevel := zap.InfoLevel
	if conf.Debug {
		logLevel = zap.DebugLevel
	}
	logCfg.Level.SetLevel(logLevel)
	logger, _ := logCfg.Build()
	log = logger.Sugar()
}

func main() {
	parseArgs()
	setupLogging()

	if conf.Seed.Scheme == "" {
		log.Warnf("no scheme specified for seed '%s', defaulting to https", conf.Seed.String())
		conf.Seed.Scheme = "https"
	}

	result := NewCrawler().Crawl(conf.Seed, conf.Timeout, conf.RateLimit)

	log.Infow("done", "result_len", len(result.links), "error_count", result.errorCount)

	if len(result.links) == 0 {
		log.Warn("no URLs processed")
		os.Exit(1)
	}

	marshaled, err := json.Marshal(result.links)
	if err != nil {
		log.Panic("failed to marshal result to JSON")
	}

	filename := fmt.Sprint(conf.Seed.Hostname(), "_sitemap.json")
	err = ioutil.WriteFile(filename, marshaled, 0664)
	if err != nil {
		log.Panic("failed to write result to file")
	}

	log.Infow("result written", "file", filename)
}
