package main

import (
    "fmt"
    "strings"
    "encoding/json"
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

const auth = "65c0f90ccf854cb5874088f30da2d82c:"

// Review stores information about a review
type Review struct {
    Author_name, Text, Posted_at, Review_id string
    Not_recommended bool
}

func main() {
    // create collector
    c := colly.NewCollector(
        colly.AllowedDomains("yelp.com", "www.yelp.com"),
    )

    // create reviews array to store review data
    reviews := make([]Review, 0, 200)

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

    // Find and get review data
    c.OnHTML(`div.not-recommended-reviews > ul.reviews > li`, func(e *colly.HTMLElement) {
        url := e.Attr("href") 
        result := strings.Contains(url, "removed_start=")
        if (!result) {
            e.Request.Visit(url)
        }
        author_name := e.ChildText("div.review-sidebar .user-display-name")
        text := e.ChildText("div.review-wrapper div.review-content p")
        posted_at := e.ChildText("div.review-wrapper div.review-content .rating-qualifier")
        review_id := e.ChildAttr("div.review--with-sidebar", "data-review-id")

        review := Review {
            Review_id: review_id,
            Author_name: author_name,
            Text: text,
            Posted_at: posted_at,
            Not_recommended: true,
        }

        reviews = append(reviews, review)
    })

    // Find and visit all next page links
    c.OnHTML("a.next", func(e *colly.HTMLElement) {
        url := e.Attr("href") 
        result := strings.Contains(url, "removed_start=")
        if (!result) {
            e.Request.Visit(url)
        }
    })
    
    // pass some headers in request
    c.OnRequest(func(r *colly.Request) {
        fmt.Println("Visiting", r.URL)
        r.Headers.Set("Proxy-Authorization", basic)
        r.Headers.Set("X-Crawlera-Profile", "desktop")
    })

    // request start page url 
    c.Visit("https://www.yelp.com/not_recommended_reviews/home-alarm-authorized-adt-dealer-lemon-grove")

    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")

    // Dump json to the standard output
    enc.Encode(reviews)
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