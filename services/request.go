package services

import (
	"fmt"
	"errors"
	"strings"
  "github.com/gocolly/colly/v2"
  "go-yelp-with-proxy/util"
  "go-yelp-with-proxy/settings"
  // "go-yelp-with-proxy/logger"
  "go.uber.org/zap"
  "io/ioutil"
  "crypto/x509"
  "encoding/base64"
  "net/http"
  "crypto/tls"
  "net/url"
  "log"

)

type ScrapyRequest struct {
	domains   []string
	proxy     string
	url       string
	log       *zap.Logger
}

func (me *ScrapyRequest) SetDomains(lst []string) {
	me.domains = lst
}

func (me *ScrapyRequest) SetProxy(proxy string) {
  me.proxy = proxy
}

func (me *ScrapyRequest) SetUrl(url string) {
	me.url = url
}

func (me *ScrapyRequest) GetProxy(name string) string {
		m := map[string]int{"url": 1, "key": 0}
		proxyDetail := strings.Split(me.proxy, "@")
		indx := m[name]

		if len(proxyDetail) >= indx - 1 {
				val := proxyDetail[indx]
				if name == "url" {
					val = "http://" + val
				}
				return val
		}
		
		panic(errors.New("Invalid proxy param"))
}

func (me *ScrapyRequest) applyProxy() *http.Transport {
    proxy := me.GetProxy("url")
    proxyURL, err := url.Parse(proxy)
    util.CheckError(err)
    
    //caCert for ssl certification
    caCert, err := ioutil.ReadFile(settings.Proxy["cert"])
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    util.CheckError(err)

    // create transport for set proxy and certificate
    return &http.Transport{
        Proxy: http.ProxyURL(proxyURL),
        TLSClientConfig: &tls.Config{
            RootCAs:      caCertPool,
        },
    }
}

func (me *ScrapyRequest) getHeaders() http.Header {
	proxyDetail := strings.Split(me.proxy, "@")
  accessKey, _ := proxyDetail[0], proxyDetail[1]
  basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(accessKey))
  return http.Header{
		"Proxy-Authorization": []string{basic},
    "X-Crawlera-Profile": []string{"desktop"},
  }
}

func (me *ScrapyRequest) Call() {
    // create collector
    c := colly.NewCollector(
        colly.AllowedDomains(me.domains...),
    )
    
    transport := me.applyProxy()
    c.WithTransport(transport)

    basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("65c0f90ccf854cb5874088f30da2d82c:"))
    c.OnHTML("html", func(e *colly.HTMLElement) {
        fmt.Println("helo")
    })

    c.OnError(func(r *colly.Response, e error) {
        log.Println("error:", e, r.Request.URL, string(r.Body))
    })

    c.Request(
        "GET",
        "https://www.yelp.com/biz/the-crack-shack-little-italy-san-diego",
        nil,
        nil,
        http.Header{"Proxy-Authorization": []string{basic},
            "X-Crawlera-Profile": []string{"desktop"},
        })

}