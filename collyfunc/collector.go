package collyfunc

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"go-yelp-with-proxy/utils"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

var YELP_USER_AGENT_STRING = []string{
	"AdsBot-Google",
	"Applebot",
	"BingPreview",
	"DeepCrawl",
	"Googlebot",
	"Googlebot-Image",
	"Googlebot-Mobile",
	"Mediapartners-Google",
	"STC-bot",
	"ScoutJet",
	"SearchmetricsBot",
	"SeznamBot",
	"TelegramBot",
	"Twitterbot",
	"Yahoo! Slurp",
	"Yandex",
	"archive.org_bot",
	"ia_archiver",
	"msnbot",
}

func getFromProxy(proxy, key string) string {
	proxyDetail := strings.Split(proxy, "@")
	accessKey, proxyUrl := proxyDetail[0], proxyDetail[1]

	ans := ""
	switch key {
	case "url":
		ans = "http://" + proxyUrl
		break
	case "key":
		ans = accessKey
		break
	}
	return ans
}
func GetColly(proxy string, scrapStatus string, requestCount int, responseBytes int) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("yelp.com", "www.yelp.com"),
		colly.Async(true),
	)
	proxyUrl := getFromProxy(proxy, "url")
	proxyURL, err := url.Parse(proxyUrl)
	utils.CheckError(err, scrapStatus)

	// create transport for set proxy and certificate
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// pass transport to collector
	c.WithTransport(transport)

	c.SetRequestTimeout(30 * time.Second)

	c.OnRequest(func(r *colly.Request) {
		requestCount += 1
		fmt.Println("Visit - ", r.URL)
		authKey := getFromProxy(proxy, "key")
		basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(authKey))
		r.Headers.Set("Proxy-Authorization", basic)
		r.Headers.Set("X-Crawlera-Profile", "desktop")
		r.Headers.Set("upgrade-insecure-requests", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("User-Agent", YELP_USER_AGENT_STRING[rand.Intn(len(YELP_USER_AGENT_STRING))])
		r.Headers.Set("authority", "www.yelp.com")
	})

	c.OnError(func(r *colly.Response, e error) {
		responseBytes += len(r.Body)
		fmt.Println("=========>", r.StatusCode)
	})

	c.OnResponse(func(r *colly.Response) {
		responseBytes += len(r.Body)
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       3 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	return c
}
