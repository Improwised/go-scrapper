package services

import (
	"fmt"
	"strings"
  "github.com/gocolly/colly/v2"
  "go-yelp-with-proxy/util"
  "go-yelp-with-proxy/settings"
  "go-yelp-with-proxy/logger"
  "go.uber.org/zap"
  "io/ioutil"
  "crypto/x509"
  "encoding/base64"
  "net/http"
  "crypto/tls"
  "net/url"

  "log"

)

type Scrapy struct {
	instance *colly.Collector
	auth string
  proxy string
  log *zap.Logger
}

func (me *Scrapy) Setup(allowedDomains []string) {
  // create collector

  me.instance = colly.NewCollector(
    colly.AllowedDomains(allowedDomains...),
  )
  me.log = logger.GetLogger()
}

func (me *Scrapy) SetProxy(proxy string) {
  me.proxy = proxy
}

func (me *Scrapy) applyProxy() {

  me.log.Info("Starting setting up proxy")

  proxyDetail := strings.Split(me.proxy, "@")
  accessKey, proxyUrl := proxyDetail[0], proxyDetail[1]

  // set proxy url
  proxyURL, err := url.Parse(proxyUrl)
  util.CheckError(err)

  me.log.Info("Proxy URL: " + proxyURL.String())

  // CA Cert for ssl certification
  caCert, err := ioutil.ReadFile(settings.Proxy["cert"])
  util.CheckError(err)
  caCertPool := x509.NewCertPool()
  caCertPool.AppendCertsFromPEM(caCert)

  // encode the auth
  me.auth = "Basic " + base64.StdEncoding.EncodeToString([]byte(accessKey))

  // create transport for set proxy and certificate
  transport := &http.Transport{
    Proxy: http.ProxyURL(proxyURL),
    TLSClientConfig: &tls.Config{
      RootCAs: caCertPool,
    },
  }

  // pass transport to collector
  me.instance.WithTransport(transport)
  me.log.Info("Proxy setup done")
}

func (me *Scrapy) Request(profileUrl string, cb func(), headers map[string]string) {
  me.applyProxy()

  // pass some headers in request
  me.instance.OnRequest(func(r *colly.Request) {
    me.log.Info("Visiting: " + r.URL.String())

    /* Default headers */
    r.Headers.Set("Proxy-Authorization", me.auth)

    /* Spider specific headers */
    for name, val := range headers {
		  r.Headers.Set(name, val)
    }

    fmt.Println(r)
  })

  // me.instance.OnResponse(func(r *colly.Response) {
  //   me.log.Info("Collected Response")
  // 	cb()
  // })

  me.instance.OnError(func(r *colly.Response, e error) {
    fmt.Println(e)
    // me.log.Error("Error: " + r.Request.URL.String())
    // // fmt.Println(r.Ctx)
    // fmt.Println(r.Request)
  })

  // request start page url
  me.instance.Visit(profileUrl)
}


func (me *Scrapy) CheckMe(profileUrl string) {
    // create collector
    fmt.Println(me.instance)
    // me.instance = colly.NewCollector(
    //     colly.AllowedDomains("yelp.com", "www.yelp.com"),
    // )

    // create reviews array to store review data

    // proxy := "http://odmarkj.crawlera.com:8010"
    // proxyURL, err := url.Parse(proxy)
    // if err != nil {
    //   panic(err)
    // }
    
    // caCert, err := ioutil.ReadFile("zyte-proxy-ca.crt")
    // caCertPool := x509.NewCertPool()
    // caCertPool.AppendCertsFromPEM(caCert)
    // if err != nil {
    //   panic(err)
    // }


    // transport := &http.Transport{
    //     Proxy: http.ProxyURL(proxyURL),
    //     TLSClientConfig: &tls.Config{
    //         RootCAs:      caCertPool,
    //     },
    // }

    // me.instance.WithTransport(transport)

    // Find and get review data
    me.instance.OnHTML(`div.not-recommended-reviews > ul.reviews > li`, func(e *colly.HTMLElement) {
      fmt.Println("review collected !")
    })

    // Find and visit all next page links
    me.instance.OnHTML("a.next", func(e *colly.HTMLElement) {
        url := e.Attr("href") 
        result := strings.Contains(url, "removed_start=")
        if (!result) {
            e.Request.Visit(url)
        }
    })
    
    // pass some headers in request
    me.instance.OnRequest(func(r *colly.Request) {
        basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("65c0f90ccf854cb5874088f30da2d82c:"))
        fmt.Println("Visiting", r.URL)
        r.Headers.Set("Proxy-Authorization", basic)
        r.Headers.Set("X-Crawlera-Profile", "desktop")
    })

    me.instance.OnError(func(r *colly.Response, e error) {
        log.Println("error:", e, r.Request.URL, string(r.Body))
        fmt.Println(e)
    })

    // request start page url 
    me.instance.Visit("https://www.yelp.com/not_recommended_reviews/home-alarm-authorized-adt-dealer-lemon-grove")
    // me.instance.Visit(profileUrl)

}


