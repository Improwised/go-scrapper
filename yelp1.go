package main

import (
    "fmt"
    "log"
    "time"
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

//proxy using for request
const auth = "65c0f90ccf854cb5874088f30da2d82c:"

// Review stores information about a review
type Reviews struct {
    Reviews []Review `json:"reviews"`
}

type Comments struct {
    Text string `json:"text"`
}

type User struct {
    Author_name string `json:"markupDisplayName"`
}

type OwnerReply []struct {
    Author_name string `json:"displayName"` 
    Text string `json:"comment"`
    Source_date string `json:"localizedDate"`
}

type PreviousReview []struct {
    Comment Comments
    Rating int `json:"rating"`
    Source_date string `json:"localizedDate"`
}

type Review struct {
    Comment Comments
    Rating int
    Photos string `json:"photosUrl"`
    Author_id string `json:"userId"`
    Review_id string `json:"id"`
    Source_date string `json:"localizedDate"`
    User User `json:"user"`
    Scraped_at, Posted_at int64
    OwnerReply OwnerReply `json:"businessOwnerReplies"`
    PreviousReview PreviousReview `json:"previousReviews"`
}

// structure for owner response for review
type OwnerReplyFomate struct {
    Author_name, Text string
    Posted_at string
}

// structure for previous review
type PreviousReviewFomate struct {
    Text string
    Rating int
    Posted_at int64
}

// Review stores information about a review
// Review stores information about a review
type ReviewFomate struct {
    Author_name, Text, Source_date, Review_id, Author_id, Photos string
    Rating int
    Scraped_at, Posted_at int64
    OwnerReply OwnerReplyFomate
    PreviousReview PreviousReviewFomate
}

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

    // create reviews array to store review data
    reviewformate := []ReviewFomate{}

    // Find and visit all next page links
    c.OnHTML("html", func(e *colly.HTMLElement) {
        // d := c.Clone()
        fmt.Println("helo")
        var reviews Reviews
        business_id := strings.Split(e.ChildAttr("meta[name=\"yelp-biz-id\"]", "content"), "\n")[0]
        url := "https://www.yelp.com/biz/" + business_id + "/review_feed?rl=en&sort_by=date_desc"
        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Set("Proxy-Authorization", basic)
        req.Header.Set("X-Crawlera-Profile", "desktop")
        res, _ := http.DefaultClient.Do(req) 
        defer res.Body.Close()
        body, _ := ioutil.ReadAll(res.Body)
        // fmt.Println(string(body))
        err := json.Unmarshal(body, &reviews) 
  
        if err != nil { 
            fmt.Println(err) 
        }
        
        for _, obj := range reviews.Reviews {
            fmt.Println(obj)
            // fmt.Println(obj.User.Author_name)

            posted_at, err := time.Parse("1/2/2006", obj.Source_date)
            checkError(err)

            review := ReviewFomate {
                Review_id: obj.Review_id,
                Author_id: obj.Author_id,
                Author_name: obj.User.Author_name,
                Text: obj.Comment.Text,
                Rating: obj.Rating,
                Source_date: obj.Source_date,
                Photos: obj.Photos,
                Posted_at: int64(posted_at.Unix()),
                Scraped_at: int64(time.Now().Unix()),
            }

            reviewformate = append(reviewformate, review)
        }

        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")

        // Dump json to the standard output
        enc.Encode(reviewformate)
    })

    c.Request(
        "GET",
        "https://www.yelp.com/biz/home-alarm-authorized-adt-dealer-lemon-grove",
        nil,
        nil,
        http.Header{"Proxy-Authorization": []string{basic},
            "X-Crawlera-Profile": []string{"desktop"},
        })

    c.OnError(func(r *colly.Response, e error) {
        log.Println("error:", e, r.Request.URL, string(r.Body))
    })
}

// func parsePage(u string, de collector, basic string) {
//     de.Visit(u)
//     // pass some headers in request
//     de.OnRequest(func(r *colly.Request) {
//         fmt.Println("Visiting", r.URL)
//         r.Headers.Set("Proxy-Authorization", basic)
//         r.Headers.Set("X-Crawlera-Profile", "desktop")
//     })
//     de.OnResponse(func(r *colly.Response) {
//         fmt.Println("yes")
//     })
        
// }

func checkError(err error) {
    if err != nil {
        if err == io.EOF {
            return
        }
        fmt.Println("Fatal error ", err.Error())
        os.Exit(1)
    }
}