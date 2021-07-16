package collyfunc

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"go-yelp-with-proxy/utils"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

var cookies []*http.Cookie

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

	c.SetRequestTimeout(100 * time.Second)

	c.OnRequest(func(r *colly.Request) {
		requestCount += 1
		fmt.Println("Visit - ", r.URL)
		authKey := getFromProxy(proxy, "key")
		basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(authKey))
		r.Headers.Set("Proxy-Authorization", basic)
		r.Headers.Set("X-Crawlera-Profile", "desktop")
		r.Headers.Set("upgrade-insecure-requests", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, 	like Gecko) Chrome/32.0.1700.72 Safari/537.36")
		r.Headers.Set("x-requested-by-react", "true")
		r.Headers.Set("x-requested-with", "XMLHttpRequest")
		if cookies != nil {
			c.SetCookies(r.URL.String(), cookies)
		}

	})

	c.OnError(func(r *colly.Response, e error) {
		responseBytes += len(r.Body)
		fmt.Println("=========>", r.StatusCode)
	})

	c.OnResponse(func(r *colly.Response) {
		responseBytes += len(r.Body)
		cookies = c.Cookies(r.Request.URL.String())
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 10,
		Delay:       3 * time.Second,
		RandomDelay: 3 * time.Second,
	})

	return c
}
