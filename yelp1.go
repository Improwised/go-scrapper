package main

import (
    "fmt"
    "log"
    "time"
    "strings"
    "strconv"
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
    Pagination struct {
        TotalResults int32 `json:"totalResults"`
        StartResult int32 `json:"startResult"`
        ResultsPerPage int32 `json:"resultsPerPage"`
    } `json:"pagination"`
}

type Pagination struct {
    TotalResults int32 
    StartResult int32 
    ResultsPerPage int32
}

type Review struct {
    Comment struct {
        Text string `json:"text"`
    }
    Rating int
    Photos string `json:"photosUrl"`
    Author_id string `json:"userId"`
    Review_id string `json:"id"`
    Source_date string `json:"localizedDate"`
    User struct {
        Author_name string `json:"markupDisplayName"`
    } `json:"user"`
    Scraped_at, Posted_at int64
    OwnerReply []struct {
        Author_name string `json:"displayName"` 
        Text string `json:"comment"`
        Source_date string `json:"localizedDate"`
    } `json:"businessOwnerReplies"`
    PreviousReview []struct {
        Comment struct {
            Text string `json:"text"`
        }
        Rating int `json:"rating"`
        Source_date string `json:"localizedDate"`
    } `json:"previousReviews"`
}

// Review stores information about a review
type ReviewFomate struct {
    Author_name, Text, Source_date, Review_id, Author_id, Photos string
    Rating int
    Scraped_at, Posted_at int64
    OwnerReply struct {
        Author_name, Text, Posted_at string
    }
    PreviousReview struct {
        Text string
        Rating int
        Posted_at int64
    }
}

type Histogram struct {
    AggregateRating struct { 
        RatingValue float32 `json:"ratingValue"`
        ReviewCount int32 `json:"reviewCount"`
    }`json:"aggregateRating"`
}

type Primary struct {
    Score float32 
    Total_reviews int32
}

type PrimaryHistogram struct {
    Primary Primary
}

func main() {
    // create collector
    c := colly.NewCollector(
        colly.AllowedDomains("yelp.com", "www.yelp.com"),
    )

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
        d := c.Clone()
        d.OnResponse(func(r *colly.Response) {
            data := &Reviews{}
            err := json.Unmarshal(r.Body, data)
            checkError(err)
            scrapReviews(data)

            pagination := Pagination {
                TotalResults: data.Pagination.TotalResults,
                StartResult: data.Pagination.StartResult,
                ResultsPerPage: data.Pagination.ResultsPerPage,
            }
            var pag Pagination 

            if (pagination != pag) {
                skip := pagination.ResultsPerPage + pagination.StartResult
                if pagination.TotalResults > skip {
                    fmt.Println(skip)
                    skipInString := strconv.FormatInt(int64(skip), 10)
                    nextPageUrl := r.Request.URL.String() + "?start=" + skipInString
                    d.Visit(nextPageUrl)
                }
            }
        })

        d.OnError(func(r *colly.Response, e error) {
            fmt.Println(e)
        })

        // pass some headers in request
        d.OnRequest(func(r *colly.Request) {
            fmt.Println("Visiting", r.URL)
            r.Headers.Set("Proxy-Authorization", basic)
            r.Headers.Set("X-Crawlera-Profile", "desktop")
        })
        business_id := strings.Split(e.ChildAttr("meta[name=\"yelp-biz-id\"]", "content"), "\n")[0]
        url := "https://www.yelp.com/biz/" + business_id + "/review_feed?rl=en&sort_by=date_desc"
        d.Visit(url)
    })

    c.OnHTML("div.main-content-wrap", func(e *colly.HTMLElement) {
        scriptData := e.ChildText("script[type=\"application/ld+json\"]")
        scriptData = scriptData[strings.Index(scriptData, "{") : strings.Index(scriptData, "}}")]
        scriptData = scriptData + "}}"
        data := Histogram{}
        err := json.Unmarshal([]byte(scriptData), &data)
        checkError(err)
        histogram := PrimaryHistogram{}
        histogram.Primary = Primary {
            Score: data.AggregateRating.RatingValue,
            Total_reviews: data.AggregateRating.ReviewCount,
        }

        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")

        // Dump json to the standard output
        enc.Encode(histogram)
    })

    // pass some headers in request
    c.OnRequest(func(r *colly.Request) {
        fmt.Println("Visiting", r.URL)
        r.Headers.Set("Proxy-Authorization", basic)
        r.Headers.Set("X-Crawlera-Profile", "desktop")
    })

    // request start page url 
    c.Visit("https://www.yelp.com/biz/home-alarm-authorized-adt-dealer-lemon-grove")

    c.OnError(func(r *colly.Response, e error) {
        log.Println("error:", e, r.Request.URL, string(r.Body))
    })
}

func scrapReviews(data *Reviews) {
    // create reviews array to store review data
    reviewformate := []ReviewFomate{}
    for _, obj := range data.Reviews {
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