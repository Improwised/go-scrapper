package main

import (
    "fmt"
    "log"
    // "bytes"
    // "strings"
    // "encoding/json"
    "github.com/gocolly/colly/v2"
    "io"
    "os"
    "io/ioutil"
    "net/http"
    "crypto/tls"
    "net/url"
    "encoding/base64"
    "crypto/x509"
)

//proxy using for request
const auth = "65c0f90ccf854cb5874088f30da2d82c:"

func main() {
    // create collector
    c := colly.NewCollector(
        colly.AllowedDomains("yelp.com", "www.yelp.com"),
    )
    
    // // create reviews array to store review data
    // reviews := []Review{}

    // set proxy url
    proxy := "http://odmarkj.crawlera.com:8010"
    proxyURL, err := url.Parse(proxy)
    checkError(err)
    
    //caCert for ssl certification
    caCert, err := ioutil.ReadFile("zyte-proxy-ca.crt")
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    checkError(err)

    // encode the auth
    basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

    // create transport for set proxy and certificate
    transport := &http.Transport{
        Proxy: http.ProxyURL(proxyURL),
        TLSClientConfig: &tls.Config{
            RootCAs:      caCertPool,
        },
    }
    
    // pass transport to collector
    c.WithTransport(transport)

    // Find and visit all next page links
    c.OnHTML("html", func(e *colly.HTMLElement) {
        fmt.Println("i am ")
    })

    c.OnResponse(func(r *colly.Response) {
        fmt.Println("i am response")
    })

    // pass some headers in request
    c.OnRequest(func(r *colly.Request) {
        fmt.Println("Visiting", r.URL)
        r.Headers.Set("Proxy-Authorization", basic)
        r.Headers.Set("Content-Type", "application/json")
        r.Headers.Set("Cache-Control", "no-cache")
        r.Headers.Set("X-Crawlera-Profile", "desktop")
    })

    jsonData := map[string]string{
        "operationName": "GetNotRecommendedReviewsProps",
        "variables": `
            { 
                "BizEncId": "nR2dFrY7VnYzJ1gtdkA5mw"
            }
        `,
        "extensions": `
            { 
                "documentId": "1cf362b8e8f9b3dae26d9f55e7204acd8355c916348a038f913845670139f60a"
            }
        `,
    }
    // jsonValue, _ := json.Marshal(jsonData)
    c.Post("https://www.yelp.com/gql/batch", jsonData)
  //   jsonData := json.Marshal(jsonData)
  //   fmt.Println(jsonData)
  //   c.POST("POST",
        // "https://www.yelp.com/gql/batch",
        // bytes.NewBuffer(jsonData),
        // nil,
        // nil)

    // request start page url 
    // c.Visit("https://www.yelp.com/biz/home-alarm-authorized-adt-dealer-lemon-grove")

    c.OnError(func(r *colly.Response, e error) {
        fmt.Println("any problem")
        fmt.Println(e)
        log.Println("error:", e, r.Request.URL, string(r.Body))
    })
}

func checkError(err error) {
    if err != nil {
        if err == io.EOF {
            return
        }
        fmt.Println("Fatal error ", err.Error())
        os.Exit(1)
    }
}