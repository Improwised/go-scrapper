package main

import (
    "fmt"
    "log"
    "strings"
    "time"
    "strconv"
    "encoding/json"
    "regexp"
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

// structure for owner response for review
type OwnerReply struct {
    Author_name, Text string
    Posted_at int64
}

// structure for previous review
type PreviousReview struct {
    Text string
    Rating int
    Posted_at int64
}

// Review stores information about a review
type Review struct {
    Author_name, Text, Source_date, Review_id, Author_id, Photos string
    Not_recommended bool
    Rating int
    Scraped_at, Posted_at int64
    OwnerReply OwnerReply
    PreviousReview PreviousReview
}

func main() {
    // create collector
    c := colly.NewCollector(
        colly.AllowedDomains("yelp.com", "www.yelp.com"),
    )

    // create reviews array to store review data
    reviews := []Review{}

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

    re := regexp.MustCompile(`regular-\s*(\d+)`) 

    // Find and get review data
    c.OnHTML(`div.not-recommended-reviews > ul.reviews > li`, func(e *colly.HTMLElement) {
        author_id := e.ChildAttr("div.review-sidebar .user-display-name", "data-hovercard-id")
        author_name := e.ChildText("div.review-sidebar .user-display-name")
        text := e.ChildText("div.review-wrapper div.review-content p")

        date := strings.Fields(e.ChildText("div.review-wrapper div.review-content .rating-qualifier"))
        source_date := date[0]

        review_id := e.ChildAttr("div.review--with-sidebar", "data-review-id")
        
        rat := re.FindStringSubmatch(e.ChildAttr(".biz-rating .i-stars", "class"))[1]
        rating, _ := strconv.Atoi(rat)

        photos := e.ChildAttr("ul.photo-box-grid div.photo-box img.photo-box-img", "data-async-src")

        posted_at, err := time.Parse("1/2/2006", source_date)
        checkError(err)

        review := Review {
            Review_id: review_id,
            Author_id: author_id,
            Author_name: author_name,
            Text: text,
            Rating: rating,
            Source_date: source_date,
            Not_recommended: true,
            Photos: photos,
            Posted_at: int64(posted_at.Unix()),
            Scraped_at: int64(time.Now().Unix()),
        }

        // if review has owner response
        var comments string
        comments = e.ChildText("div.review-wrapper div.biz-owner-reply span.bullet-after")

        if comments != "" {
            source_date := e.ChildText("div.biz-owner-reply span.bullet-after")
            posted_at, err := time.Parse("1/2/2006", source_date)
            checkError(err)

            response := OwnerReply {
                Author_name : strings.Replace(e.ChildText("div.biz-owner-reply-header strong"), "Comment from ", "", -1),
                Text : e.ChildText("span.js-content-toggleable.hidden"),
                Posted_at : int64(posted_at.Unix()),
            }

            review.OwnerReply = response
        }

        // if review has previous review
        var previous string
        previous = e.ChildText("div.review-wrapper div.previous-review span.js-expandable-comment span.js-content-toggleable")

        if previous != "" {
            date := strings.Fields(e.ChildText("div.review-wrapper div.previous-review .rating-qualifier"))
            source_date := date[0]
            posted_at, err := time.Parse("1/2/2006", source_date)
            checkError(err)

            rat := re.FindStringSubmatch(e.ChildAttr(".previous-review .biz-rating .i-stars", "class"))[1]
            rating, _ := strconv.Atoi(rat)

            previous_review := PreviousReview {
                Text : previous,
                Rating: rating,
                Posted_at : int64(posted_at.Unix()),
            }

            review.PreviousReview = previous_review
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

    c.OnError(func(r *colly.Response, e error) {
        log.Println("error:", e, r.Request.URL, string(r.Body))
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