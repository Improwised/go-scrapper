package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/spf13/cobra"
)

// Define required Structs
type Spider struct {
	ProfileKey       string   `json:"profile_key"`
	BusinessName     string   `json:"business_name"`
	LastReviewHashes []string `json:"last_review_hashes"`
	BusinessID       int      `json:"business_id"`
	ClientID         int      `json:"client_id"`
	BatchID          int      `json:"batch_id"`
	TaskID           int      `json:"task_id"`
	Persona          struct {
		AdditionalCookies interface{} `json:"additional_cookies"`
		Proxy             string      `json:"proxy"`
		OtherProxies      []string    `json:"other_proxies"`
	} `json:"persona"`
	Address struct {
		City   string `json:"city"`
		State  string `json:"state"`
		Street string `json:"street"`
		Zip    string `json:"zip"`
	} `json:"address"`
	filename string
}

type Reviews struct {
	Reviews []Review `json:"reviews"`
}

type Review struct {
	Comment struct {
		Text string `json:"text"`
	}
	Rating      int
	Photos      string `json:"photosUrl"`
	Author_id   string `json:"userId"`
	Review_id   string `json:"id"`
	Source_date string `json:"localizedDate"`
	User        struct {
		Author_name string `json:"markupDisplayName"`
	} `json:"user"`
	Scraped_at, Posted_at int64
	OwnerReply            []struct {
		Author_name string `json:"displayName"`
		Text        string `json:"comment"`
		Source_date string `json:"localizedDate"`
	} `json:"businessOwnerReplies"`
	PreviousReview []struct {
		Comment struct {
			Text string `json:"text"`
		}
		Rating      int    `json:"rating"`
		Source_date string `json:"localizedDate"`
	} `json:"previousReviews"`
}

type OwnerReply struct {
	Author_name, Text string
	Posted_at         int64
}

// structure for previous review
type PreviousReview struct {
	Text      string
	Rating    int
	Posted_at int64
}

// Review stores information about a review
type ReviewFomate struct {
	Author_name, Text, Source_date string
	Review_id, Author_id, Photos   string
	Not_recommended                bool
	Rating                         int
	Scraped_at, Posted_at          int64
	OwnerReply                     OwnerReply
	PreviousReview                 PreviousReview
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
	fmt.Println("Hello test.")

	var cmd = &cobra.Command{
		Use:   "yelp",
		Short: "Run spider yelp",
		Long:  "Run spider yelp",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			additional_args := cmd.Flag("additional-args").Value.String()
			op := cmd.Flag("output").Value.String()
			yelpSpiderRun(additional_args, op)
		},
	}

	// Setup arguments
	cmd.PersistentFlags().StringP("additional-args", "a", "", "NAME=VALUE as additional Arguments.")
	cmd.PersistentFlags().StringP("output", "o", "", "output filename.")

	// Execute command and handle Error
	if err := cmd.Execute(); err != nil {
		panic(err)
	}

}

func setPlace(args string, sp *Spider) {
	additionalArgs := strings.Split(args, "=")
	if len(additionalArgs) >= 2 {
		_, p := additionalArgs[0], additionalArgs[1]
		place, err := base64.StdEncoding.DecodeString(p)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(place, sp)
		if err != nil {
			panic(err)
		}
	} else {
		panic(errors.New("Invalid Additional Arguments."))
	}
}

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
	fmt.Println("Proxy(" + key + "): " + ans)
	return ans
}

func getColly(proxy string) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("yelp.com", "www.yelp.com"),
	)
	proxyUrl := getFromProxy(proxy, "url")
	fmt.Println(proxyUrl)
	proxyURL, err := url.Parse(proxyUrl)
	checkError(err)

	//caCert for ssl certification
	caCert, err := ioutil.ReadFile("zyte-proxy-ca.crt")
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	checkError(err)

	// create transport for set proxy and certificate
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	// pass transport to collector
	c.WithTransport(transport)

	c.OnRequest(func(r *colly.Request) {
		// fmt.Println("Visiting", r.URL)
		authKey := getFromProxy(proxy, "key")
		basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(authKey))
		r.Headers.Set("Proxy-Authorization", basic)
		r.Headers.Set("X-Crawlera-Profile", "desktop")
	})

	c.OnError(func(r *colly.Response, e error) {
		// log.Println("error:", e, r.Request.URL, string(r.Body))
	})

	return c
}

func yelpSpiderRun(args, op string) {
	spider := &Spider{filename: op}
	setPlace(args, spider)

	if spider.ProfileKey == "" {
		fmt.Println("We are not supporting business without profile key as of now.")
		os.Exit(1)
	}
	reviews := []ReviewFomate{}
	rev_counter := 0
	non_counter := 0

	var profileColly = getColly(spider.Persona.Proxy)
	var reviewCollector = getColly(spider.Persona.Proxy)
	var nonRevCollector = getColly(spider.Persona.Proxy)
	var nonRevCollector2 = getColly(spider.Persona.Proxy)

	var wgA sync.WaitGroup
	var wgB sync.WaitGroup
	var business_id string
	wgB.Add(1)
	fmt.Println("Initialize wait group.")

	// Find and visit all next page links
	profileColly.OnHTML("html", func(e *colly.HTMLElement) {
		fmt.Println("Collected Html")

		business_id = strings.Split(e.ChildAttr("meta[name=\"yelp-biz-id\"]", "content"), "\n")[0]
		wgB.Done()
		RevUrl := "https://www.yelp.com/biz/" + business_id + "/review_feed?rl=en&sort_by=date_desc"
		fmt.Println(RevUrl)

		jsonStr := e.ChildText("script[type=\"application/ld+json\"]")

		// re := regexp.MustCompile("\\$\\{(.*?)\\}")
		re := regexp.MustCompile("\"reviewCount\":(\\d*)")
		match := re.FindStringSubmatch(jsonStr)
		reviewCount, err := strconv.Atoi(match[1])
		if err != nil {
			panic(err)
		}
		fmt.Println(reviewCount)

		if strings.Contains(jsonStr, "reviewCount") {
			fmt.Println("Found")
		} else {
			fmt.Println("Not Found !")
		}
		for i := 0; i < reviewCount; i += 10 {
			wgA.Add(1)
			rev_counter += 1
			fmt.Println("Counters", rev_counter, non_counter)
			go reviewCollector.Visit(RevUrl + "&start=" + strconv.Itoa(i))
		}

	})

	// collect histogram data
    profileColly.OnHTML("div.main-content-wrap", func(e *colly.HTMLElement) {
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
        fmt.Println("Histogram", histogram)
    })

	profileColly.OnResponse(func(r *colly.Response) {
		go func() {
			wgB.Wait()
			// Non Recommended Reviews
			NrRevUrl, err := url.Parse("/not_recommended_reviews/" + business_id)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("URL ==>")
			nrRevURL := r.Request.URL.ResolveReference(NrRevUrl)
			fmt.Println(nrRevURL)
			nonRevCollector.Visit(nrRevURL.String())
		}()
	})

	reviewCollector.OnResponse(func(r *colly.Response) {
		data := &Reviews{}
		err := json.Unmarshal(r.Body, data)
		checkError(err)
		for _, obj := range data.Reviews {
			posted_at, err := time.Parse("1/2/2006", obj.Source_date)
			checkError(err)

			review := ReviewFomate{
				Review_id:   obj.Review_id,
				Author_id:   obj.Author_id,
				Author_name: obj.User.Author_name,
				Text:        obj.Comment.Text,
				Rating:      obj.Rating,
				Source_date: obj.Source_date,
				Photos:      obj.Photos,
				Posted_at:   int64(posted_at.Unix()),
				Scraped_at:  int64(time.Now().Unix()),
			}

			reviews = append(reviews, review)
		}
		rev_counter -= 1
		fmt.Println("Counters", rev_counter, non_counter)
		wgA.Done()
	})

	nonRevCollector.OnHTML("h3", func(e *colly.HTMLElement) {
		if strings.Contains(e.Text, "recommended") {
			re := regexp.MustCompile("(\\d+)")
			match := re.FindStringSubmatch(e.Text)
			reviewCount, err := strconv.Atoi(match[1])
			if err != nil {
				panic(err)
			}
			fmt.Println("Non Recoommended Reviews -- ", reviewCount)
			fmt.Println(e.Request.URL)
			for i := 0; i < reviewCount; i += 10 {
				wgA.Add(1)
				non_counter += 1
				fmt.Println("Counters", rev_counter, non_counter)
				visitingUrl := e.Request.URL.String() + "?start=" + strconv.Itoa(i)
				go nonRevCollector2.Visit(visitingUrl)
			}
		}
	})

	nonRevCollector2.OnResponse(func(r *colly.Response) {
		non_counter -= 1
		fmt.Println("Counters", rev_counter, non_counter)
		wgA.Done()
	})

	nonRevCollector2.OnHTML(`div.not-recommended-reviews > ul.reviews > li`, func(e *colly.HTMLElement) {
		author_id := e.ChildAttr("div.review-sidebar .user-display-name", "data-hovercard-id")
		author_name := e.ChildText("div.review-sidebar .user-display-name")
		text := e.ChildText("div.review-wrapper div.review-content p")

		date := strings.Fields(e.ChildText("div.review-wrapper div.review-content .rating-qualifier"))
		source_date := date[0]

		review_id := e.ChildAttr("div.review--with-sidebar", "data-review-id")

		re := regexp.MustCompile(`regular-\s*(\d+)`)
		rat := re.FindStringSubmatch(e.ChildAttr(".biz-rating .i-stars", "class"))[1]
		rating, _ := strconv.Atoi(rat)

		photos := e.ChildAttr("ul.photo-box-grid div.photo-box img.photo-box-img", "data-async-src")

		posted_at, err := time.Parse("1/2/2006", source_date)
		checkError(err)

		review := ReviewFomate{
			Review_id:       review_id,
			Author_id:       author_id,
			Author_name:     author_name,
			Text:            text,
			Rating:          rating,
			Source_date:     source_date,
			Not_recommended: true,
			Photos:          photos,
			Posted_at:       int64(posted_at.Unix()),
			Scraped_at:      int64(time.Now().Unix()),
		}

		// if review has owner response
		var comments string
		comments = e.ChildText("div.review-wrapper div.biz-owner-reply span.bullet-after")

		if comments != "" {
			source_date := e.ChildText("div.biz-owner-reply span.bullet-after")
			posted_at, err := time.Parse("1/2/2006", source_date)
			checkError(err)

			response := OwnerReply{
				Author_name: strings.Replace(e.ChildText("div.biz-owner-reply-header strong"), "Comment from ", "", -1),
				Text:        e.ChildText("span.js-content-toggleable.hidden"),
				Posted_at:   int64(posted_at.Unix()),
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

			previous_review := PreviousReview{
				Text:      previous,
				Rating:    rating,
				Posted_at: int64(posted_at.Unix()),
			}

			review.PreviousReview = previous_review
		}

		reviews = append(reviews, review)
	})

	reviewCollector.OnError(func(r *colly.Response, e error) {
		wgA.Done()
		// fmt.Println("Retrying", r.Request.URL.String())
		// go reviewCollector.Visit(r.Request.URL.String())
	})

	nonRevCollector2.OnError(func(r *colly.Response, e error) {
		wgA.Done()
		// fmt.Println("Retrying", r.Request.URL.String())
		// go nonRevCollector2.Visit(r.Request.URL.String())
	})

	// request start page url
	profileColly.Visit(spider.ProfileKey)

	wgA.Wait()
	fmt.Println("FINALLY DONE ! ", len(reviews))
	fmt.Println("Couters -- ", rev_counter, non_counter)
	// out, err := json.Marshal(reviews)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(out))
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
