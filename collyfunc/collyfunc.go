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

var USER_AGENT_STRINGS = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.8; rv:43.0) Gecko/20100101 Firefox/43.0",
	"Mozilla/5.0 (X11; Linux i586; rv:31.0) Gecko/20100101 Firefox/31.0",
	"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:31.0) Gecko/20130401 Firefox/31.0",
	"Mozilla/5.0 (Windows NT 5.1; rv:31.0) Gecko/20100101 Firefox/31.0",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:25.0) Gecko/20100101 Firefox/25.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.6; rv:25.0) Gecko/20100101 Firefox/25.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux i686; rv:11.0) Gecko/20100101 Firefox/11.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_1) AppleWebKit/537.36 (KHTML, like Gecko) ",
	"Chrome/41.0.2227.1 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_2) AppleWebKit/537.36 (KHTML, like Gecko) ",
	"Chrome/36.0.1944.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10; rv:33.0) Gecko/20100101 Firefox/33.0",
	"Mozilla/5.0 (Windows NT 6.3; rv:36.0) Gecko/20100101 Firefox/36.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_3) AppleWebKit/537.75.14 (KHTML, like Gecko) ",
	"Version/7.0.3 Safari/7046A194A",
	"Mozilla/5.0 (X11; U; Linux x86_64; en-us) AppleWebKit/531.2+ (KHTML, like Gecko) Version/5.0 ",
	"Safari/531.2+",
	"Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.1; WOW64; Trident/6.0)",
	"Opera/9.80 (X11; Linux i686; Ubuntu/14.10) Presto/2.12.388 Version/12.16",
	"Opera/12.0(Windows NT 5.2;U;en)Presto/22.9.168 Version/12.00",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_5) AppleWebKit/537.36 (KHTML, like Gecko) ",
	"Chrome/43.0.2357.130 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_5) AppleWebKit/537.36 (KHTML, like Gecko) ",
	"Chrome/44.0.2395.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_1) AppleWebKit/537.36 (KHTML, like Gecko) ",
	"Chrome/41.0.2227.1 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_2) AppleWebKit/537.36 (KHTML, like Gecko) ",
	"Chrome/36.0.1944.0 Safari/537.36",
}

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
		colly.IgnoreRobotsTxt(),
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
		ExpectContinueTimeout: 4 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	// pass transport to collector
	c.WithTransport(transport)

	c.SetRequestTimeout(200 * time.Second)

	c.OnRequest(func(r *colly.Request) {
		requestCount += 1
		fmt.Println("Visit - ", r.URL)
		authKey := getFromProxy(proxy, "key")
		basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(authKey))
		r.Headers.Set("Proxy-Authorization", basic)
		r.Headers.Set("X-Crawlera-Profile", "desktop")
		r.Headers.Set("upgrade-insecure-requests", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("User-Agent", USER_AGENT_STRINGS[rand.Intn(len(USER_AGENT_STRINGS))])
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
		Delay:       3 * time.Second,
		RandomDelay: 3 * time.Second,
	})

	return c
}

func GetReviewColly(proxy string, scrapStatus string, requestCount int, responseBytes int) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("yelp.com", "www.yelp.com"),
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
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
		ExpectContinueTimeout: 4 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	// pass transport to collector
	c.WithTransport(transport)

	c.SetRequestTimeout(200 * time.Second)

	c.OnRequest(func(r *colly.Request) {
		requestCount += 1
		fmt.Println("Visit - ", r.URL)
		authKey := getFromProxy(proxy, "key")
		basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(authKey))
		r.Headers.Set("Proxy-Authorization", basic)
		r.Headers.Set("X-Crawlera-Profile", "desktop")
		r.Headers.Set("upgrade-insecure-requests", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("User-Agent", USER_AGENT_STRINGS[rand.Intn(len(USER_AGENT_STRINGS))])
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
