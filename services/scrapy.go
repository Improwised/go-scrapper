package services

import (
	"fmt"
	"log"
	"strings"
  "github.com/gocolly/colly/v2"
  "go-yelp-with-proxy/util"
  "go-yelp-with-proxy/settings"
  "io/ioutil"
  "crypto/x509"
  "encoding/base64"
  "net/http"
  "crypto/tls"
  "net/url"
)

type Scrapy struct {
	instance *colly.Collector
	auth string
}

func (me *Scrapy) Setup(allowedDomains []string) {
  // create collector
  me.instance = colly.NewCollector(
    colly.AllowedDomains(allowedDomains...),
  )
}

func (me *Scrapy) SetProxy(proxy string) {

  proxyDetail := strings.Split(proxy, "@")
  accessKey, proxyUrl := proxyDetail[0], proxyDetail[1]

  // set proxy url
  proxyURL, err := url.Parse(proxyUrl)
  util.CheckError(err)

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
}

func (me *Scrapy) Request(profileUrl string) {

  // pass some headers in request
  me.instance.OnRequest(func(r *colly.Request) {
    fmt.Println("Visiting", r.URL)
    r.Headers.Set("Proxy-Authorization", me.auth)
    r.Headers.Set("X-Crawlera-Profile", "desktop")
  })

  me.instance.OnError(func(r *colly.Response, e error) {
    log.Println("error:", e, r.Request.URL, string(r.Body))
  })

  // request start page url
  me.instance.Visit(profileUrl)
}
