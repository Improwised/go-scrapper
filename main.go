package main

import (
	"bytes"
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

type HistogramFormat struct {
	AggregateRating struct {
		RatingValue float32 `json:"ratingValue"`
		ReviewCount int32   `json:"reviewCount"`
	} `json:"aggregateRating"`
}

type Primary struct {
	Score         float32
	Total_reviews int32
}

type Histogram struct {
	Primary Primary
}

func main() {
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
	fmt.Println(args)
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
	// fmt.Println("Proxy(" + key + "): " + ans)
	return ans
}

func getColly(proxy string) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("yelp.com", "www.yelp.com"),
	)
	proxyUrl := getFromProxy(proxy, "url")
	// fmt.Println(proxyUrl)
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
		fmt.Println("Visit - ", r.URL)
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

func updateAndPrintCnt(r *int, c *int, t int, a int) {
	if t == 1 {
		*r += a
	}
	if t == 2 {
		*c += a
	}
	fmt.Println("Counters", *r, *c)
}

var (
	reviews     []ReviewFomate
	histogram   Histogram
	spider      *Spider
	rev_counter int
	non_counter int
	err_counter int
	business_id string
)

func yelpSpiderRun(args, op string) {

	// Initialize variables
	spider := &Spider{filename: op}
	setPlace(args, spider)

	if spider.ProfileKey == "" {
		fmt.Println("We are not supporting business without profile key as of now.")
		os.Exit(1)
	}

	// Profile URL Call
	var wg sync.WaitGroup
	wg.Add(1)
	fmt.Println(">>>>>>>>>>>> ADD - initial")
	go callProfileURL(spider, &wg)
	fmt.Println("Waiting...")
	wg.Wait()
	fmt.Println("Profile Call done ! -- Count", len(reviews))
	dumpReviews(spider)
}

func callProfileURL(spider *Spider, wg *sync.WaitGroup) {
	profile := getColly(spider.Persona.Proxy)
	profile.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
		wg.Done() // for histogram
	})
	profile.OnHTML(`html`, func(e *colly.HTMLElement) {
		fmt.Println("Response - ", e.Request.URL.String())

		// Collect Business ID
		businessId := strings.Split(e.ChildAttr("meta[name=\"yelp-biz-id\"]", "content"), "\n")[0]
		fmt.Println("Business ID:", businessId)

		// ===================================
		// Collecting Histogram
		// ===================================
		// scriptData := e.ChildText("script[type=\"application/ld+json\"]")
		// scriptData = scriptData[strings.Index(scriptData, "{"):strings.Index(scriptData, "}}")]
		// scriptData = scriptData + "}}"
		// data := HistogramFormat{}
		// err := json.Unmarshal([]byte(scriptData), &data)
		// checkError(err)
		// histogram := Histogram{}
		// histogram.Primary = Primary{
		// 	Score:         data.AggregateRating.RatingValue,
		// 	Total_reviews: data.AggregateRating.ReviewCount,
		// }
		// fmt.Println("Histogram", histogram)

		// ===================================
		// Non Recommanded Review Scrap
		// ===================================

		// Prepare Non Recommanded URL
		nonUrl, err := url.Parse("/not_recommended_reviews/" + businessId)
		if err != nil {
			log.Fatal(err)
		}
		nonRevURL := e.Request.URL.ResolveReference(nonUrl)

		// Fist visit to non recommanded URL
		fmt.Println("Non Recommmanded URL:", nonRevURL)
		fmt.Println(">>>>>>>>>>>> ADD - non recommended first")
		wg.Add(1)
		go nonRecommandedReviewUrlCall(spider, wg, nonRevURL.String())

		// ===================================
		// Normal Review Scrap
		// ===================================

		// Prepare URL
		RevUrl := "https://www.yelp.com/biz/" + businessId + "/review_feed?rl=en&sort_by=date_desc"

		// Collect Review Count
		jsonStr := e.ChildText("script[type=\"application/ld+json\"]")
		re := regexp.MustCompile("\"reviewCount\":(\\d*)")
		match := re.FindStringSubmatch(jsonStr)
		reviewCount, err := strconv.Atoi(match[1])
		if err != nil {
			panic(err)
		}
		fmt.Println("Normal Reviews:", reviewCount)
		fmt.Println("URL", RevUrl)

		// Call all pages.
		for i := 0; i < reviewCount; i += 10 {
			wg.Add(1)
			go normalReview(spider, wg, RevUrl+"&start="+strconv.Itoa(i))
		}
		fmt.Println(">>>>>>>>>>>> DONE - initial")
		wg.Done()
	})
	profile.Visit(spider.ProfileKey)
}

func normalReview(spider *Spider, wg *sync.WaitGroup, link string) {
	linkCall := getColly(spider.Persona.Proxy)
	linkCall.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
		ilink := r.Request.URL.String()
		fmt.Println("URL Error:", ilink)
		wg.Done()
	})
	linkCall.OnResponse(func(r *colly.Response) {
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
			rev_counter += 1
		}
		fmt.Println("Non Counter", rev_counter)
		wg.Done()
	})
	linkCall.Visit(link)
}

func nonRecommandedReviewUrlCall(spider *Spider, wg *sync.WaitGroup, link string) {
	linkCall := getColly(spider.Persona.Proxy)
	linkCall.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
		wg.Done()
	})
	linkCall.OnHTML(`html`, func(e *colly.HTMLElement) {
		fmt.Println("Response - ", e.Request.URL.String())

		nonReviewCount := 0

		for _, v := range e.ChildTexts("h3") {
			fmt.Println("H3 =", v)
			if strings.Contains(v, "recommended") {
				re := regexp.MustCompile("(\\d+)")
				match := re.FindStringSubmatch(v)
				count, err := strconv.Atoi(match[1])
				if err != nil {
					panic(err)
				}
				nonReviewCount = count
				fmt.Println("OK...", nonReviewCount)
			}
		}

		fmt.Println("Non recommanded Reviews", nonReviewCount)
		fmt.Println("Link", e.Request.URL.String())

		for i := 0; i < nonReviewCount; i += 10 {
			fmt.Println(">>>>>>>>>>>> ADD - non rec follow up")
			wg.Add(1)
			visitingUrl := e.Request.URL.String() + "?not_recommended_start=" + strconv.Itoa(i)
			go nonRecommandedReviewUrlCallFollowup(spider, wg, visitingUrl)
		}
		fmt.Println(">>>>>>>>>>>> DONE - non rec first")
		wg.Done()
	})
	linkCall.Visit(link)
}

func nonRecommandedReviewUrlCallFollowup(spider *Spider, wg *sync.WaitGroup, link string) {
	linkCall := getColly(spider.Persona.Proxy)
	linkCall.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
		wg.Done()
	})
	linkCall.OnHTML(`html`, func(e *colly.HTMLElement) {
		nonReviewCount := len(e.ChildTexts(`div.not-recommended-reviews > ul.reviews > li`))
		fmt.Println("Review Count", nonReviewCount, e.Request.URL.String())
		wg.Add(nonReviewCount)
		wg.Done()
		fmt.Println("Non Counter", non_counter)
	})
	linkCall.OnHTML(`div.not-recommended-reviews > ul.reviews > li`, func(e *colly.HTMLElement) {
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
		non_counter += 1
		wg.Done()
	})
	linkCall.Visit(link)
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

func dumpReviews(spider *Spider) {
	for _, v := range reviews {
		fmt.Println(v)
		n, err := WriteDataToFileAsJSON(v, spider.filename)
		if err != nil {
			panic(err)
		}
		print("Written Count", n)
	}
}

func WriteDataToFileAsJSON(data interface{}, filedir string) (int, error) {
	//write data as buffer to json encoder
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	// encoder.SetIndent("", "\t")

	err := encoder.Encode(data)
	if err != nil {
		return 0, err
	}
	file, err := os.OpenFile(filedir, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return 0, err
	}
	n, err := file.Write(buffer.Bytes())
	if err != nil {
		return 0, err
	}
	return n, nil
}
