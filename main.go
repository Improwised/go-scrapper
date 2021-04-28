package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	Rating int
	Photos []struct {
		Src string `json:"src"`
	} `json:"photos"`
	Author_id   string `json:"userId"`
	Review_id   string `json:"id"`
	Source_date string `json:"localizedDate"`
	User        struct {
		Author_name string `json:"markupDisplayName"`
	} `json:"user"`
	OwnerReply []struct {
		Author_name struct {
			Name string `json:"displayName"`
		} `json:"owner"`
		Text        string `json:"comment"`
		Source_date string `json:"localizedDate"`
	} `json:"businessOwnerReplies"`
	PreviousReview []struct {
		Comment struct {
			Text string `json:"text"`
		}
		User struct {
			Author_name string `json:"markupDisplayName"`
		} `json:"user"`
		Photos []struct {
			Src string `json:"src"`
		} `json:"photos"`
		Author_id   string `json:"userId"`
		Review_id   string `json:"id"`
		Rating      int    `json:"rating"`
		Source_date string `json:"localizedDate"`
		OwnerReply  []struct {
			Author_name struct {
				Name string `json:"displayName"`
			} `json:"owner"`
			Text        string `json:"comment"`
			Source_date string `json:"localizedDate"`
		} `json:"businessOwnerReplies"`
	} `json:"previousReviews"`
}

type OwnerReply struct {
	Author_name string `json:"author_name,omitempty"`
	Text        string `json:"text,omitempty"`
	Posted_at   string `json:"posted_at,omitempty"`
}

// Review stores information about a review
type ReviewFomate struct {
	Parent_id       string       `json:"parent_id,omitempty"`
	Author_name     string       `json:"author_name,omitempty"`
	Text            string       `json:"text,omitempty"`
	Source_date     string       `json:"source_date,omitempty"`
	Review_id       string       `json:"review_id,omitempty"`
	Author_id       string       `json:"author_id,omitempty"`
	Photos          []string     `json:"photos,omitempty"`
	Not_recommended bool         `json:"not_recommended,omitempty"`
	Rating          int          `json:"rating,omitempty"`
	Scraped_at      int64        `json:"scraped_at,omitempty"`
	Posted_at       int64        `json:"posted_at,omitempty"`
	OwnerReply      []OwnerReply `json:"responses,omitempty"`
	ReviewHash      string       `json:"review_hash"`
}

type HistogramFormat struct {
	AggregateRating struct {
		RatingValue float32 `json:"ratingValue"`
		ReviewCount int32   `json:"reviewCount"`
	} `json:"aggregateRating"`
}

type Primary struct {
	Score         float32 `json:"score"`
	Total_reviews int32   `json:"total_revews"`
}

type Histogram struct {
	Primary Primary `json:"primary"`
}

type Meta struct {
	Histogram          Histogram `json:"histogram"`
	Profile_key        string    `json:"profile_key"`
	Start_time         string    `json:"start_time"`
	Finish_time        string    `json:"finish_time"`
	Scraping_status    string    `json:"scraping_status"`
	Item_scraped_count int       `json:"item_scraped_count"`
	Request_count      int       `json:"downloader/request_count"`
	Response_bytes     int       `json:"downloader/response_bytes"`
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
			setvar := cmd.Flag("setvar").Value.String()
			yelpSpiderRun(additional_args, op, setvar)
		},
	}

	// Setup arguments
	cmd.PersistentFlags().StringP("additional-args", "a", "", "NAME=VALUE as additional Arguments.")
	cmd.PersistentFlags().StringP("output", "o", "", "output filename.")
	cmd.PersistentFlags().StringP("setvar", "s", "", "NAME=VALUE as setting variable .")

	// Execute command and handle Error
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func setPlace(args string, sp *Spider) {
	additionalArgs := strings.Split(args, "=")
	fmt.Println(args)
	if len(additionalArgs) >= 2 {
		p := strings.Join(additionalArgs[1:], "=")
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
	return ans
}

func getColly(proxy string) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains("yelp.com", "www.yelp.com"),
		colly.Async(true),
	)
	proxyUrl := getFromProxy(proxy, "url")
	proxyURL, err := url.Parse(proxyUrl)
	checkError(err)

	// create transport for set proxy and certificate
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// pass transport to collector
	c.WithTransport(transport)

	c.OnRequest(func(r *colly.Request) {
		requestCount += 1
		fmt.Println("Visit - ", r.URL)
		authKey := getFromProxy(proxy, "key")
		basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(authKey))
		r.Headers.Set("Proxy-Authorization", basic)
		r.Headers.Set("X-Crawlera-Profile", "desktop")
	})

	c.OnError(func(r *colly.Response, e error) {
		responseBytes += len(r.Body)
		fmt.Println("=========>", r.StatusCode)
	})

	c.OnResponse(func(r *colly.Response) {
		responseBytes += len(r.Body)
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       2 * time.Second,
	})

	return c
}

var (
	reviews              []ReviewFomate
	histogram            Histogram
	spider               *Spider
	rev_counter          int
	non_counter          int
	err_counter          int
	minimal_review_count int
	item_scraped_count   int
	business_id          string
	start_time           string
	finish_time          string
	scrapStatus          string
	requestCount         int
	responseBytes        int
	mu                   sync.Mutex
)

func yelpSpiderRun(args, op, sval string) {

	// Initialize variables
	spider := &Spider{filename: op}
	setPlace(args, spider)

	if spider.ProfileKey == "" {
		fmt.Println("We are not supporting business without profile key as of now.")
		os.Exit(1)
	}

	// Profile URL Call
	var wg sync.WaitGroup
	wg.Add(1) // add PROFILE call
	start_time = time.Now().UTC().Format("2006-01-02 15:04:05")
	callProfileURL(spider, &wg)
	fmt.Println("Waiting...")
	wg.Wait() // Wait for completing all calls
	finish_time = time.Now().UTC().Format("2006-01-02 15:04:05")
	fmt.Println("Profile Call done ! -- Count", len(reviews))
	item_scraped_count = len(reviews)
	if (len(reviews) > 0) {
		scrapStatus = "SUCCESS_SCRAPED"
		if (scrapStatus == "") {
			scrapStatus = "SCRAPE_FAILED"
		}
	}  
	dumpReviews(spider.filename)
	dumpMetaData(spider)
	fmt.Println("Scrapping - ", scrapStatus)
}

func callProfileURL(spider *Spider, wg *sync.WaitGroup) {
	profile := getColly(spider.Persona.Proxy)
	profile.OnError(func(r *colly.Response, e error) {
		fmt.Println("Status ", r.StatusCode)
		if r.StatusCode == 404 {
			scrapStatus = "PAGE_NOT_FOUND"
		}
		if r.StatusCode == 503 {
			scrapStatus = "SCRAPE_FAILED"
		}
		if (len(r.Body) == 0 && r.StatusCode == 0) {
			if strings.Contains(e.Error(), "Client.Timeoutome") {
				scrapStatus = "TIMEOUT"
			}
		}
		log.Println("error:", e, r.Request.URL, string(r.Body))
		wg.Done() // done PROFILE call [failed]
	})
	profile.OnHTML(`html`, func(e *colly.HTMLElement) {
		fmt.Println("Response - ", e.Request.URL.String())

		// Collect Business ID
		businessId := strings.Split(e.ChildAttr("meta[name=\"yelp-biz-id\"]", "content"), "\n")[0]
		fmt.Println("Business ID:", businessId)

		// ===================================
		// Collecting Histogram
		// ===================================
		scriptData := e.ChildText("script[type=\"application/ld+json\"]")
		if len(scriptData) >= 1 {
			scriptData = scriptData[strings.Index(scriptData, "{"):strings.Index(scriptData, "}{")]
			scriptData = scriptData + "}"
			data := HistogramFormat{}
			err := json.Unmarshal([]byte(scriptData), &data)
			checkError(err)
			histogram.Primary = Primary{
				Score:         data.AggregateRating.RatingValue,
				Total_reviews: data.AggregateRating.ReviewCount,
			}

			fmt.Println("Histogram:", histogram)
		}
		
		// ===================================
		// Normal Review Scrap
		// ===================================

		// Prepare URL
		RevUrl := "https://www.yelp.com/biz/" + businessId + "/review_feed?rl=en&sort_by=date_desc"

		// Collect Review Count
		jsonStr := e.ChildText("script[type=\"application/ld+json\"]")
		re := regexp.MustCompile("\"reviewCount\":(\\d*)")
		match := re.FindStringSubmatch(jsonStr)
		if len(match) >= 2 {
			reviewCount, err := strconv.Atoi(match[1])
			if err != nil {
				panic(err)
			}
			fmt.Println("Normal Reviews:", reviewCount)
			minimal_review_count = reviewCount
			// Call all pages.
			var reviewCollector = normalReview(spider, wg)
			for i := 0; i < reviewCount; i += 10 {
				wg.Add(1) // add REVIEW call
				reviewCollector.Visit(RevUrl + "&start=" + strconv.Itoa(i))
			}
		} 

		// ===================================
		// Non Recommanded Review Scrap
		// ===================================

		// Prepare Non Recommanded URL
		nonUrl, err := url.Parse("/not_recommended_reviews/" + businessId)
		if err != nil {
			log.Fatal(err)
		}
		nonRevURL := e.Request.URL.ResolveReference(nonUrl)

		wg.Add(1) // add NON_RECOMMENDED_ONCE call

		// Fist visit to non recommanded URL
		nonRecommandedReviewUrlCall(spider, wg, nonRevURL.String())

		wg.Done() // done PROFILE call [success]
	})
	profile.Visit(spider.ProfileKey)
}

func normalReview(spider *Spider, wg *sync.WaitGroup) *colly.Collector {
	linkCall := getColly(spider.Persona.Proxy)
	linkCall.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
		ilink := r.Request.URL.String()
		fmt.Println("URL Error:", ilink)
		wg.Done() // done REVIEW call [failed]
	})
	linkCall.OnResponse(func(r *colly.Response) {
		data := &Reviews{}
		err := json.Unmarshal(r.Body, data)
		checkError(err)
		for _, obj := range data.Reviews {
			posted_at, err := time.Parse("1/2/2006", obj.Source_date)
			checkError(err)
			var photo []string
			for _, photoObj := range obj.Photos {
				photo = append(photo, photoObj.Src)
			}

			review := ReviewFomate{
				Review_id:   obj.Review_id,
				Author_id:   obj.Author_id,
				Author_name: obj.User.Author_name,
				Text:        html.UnescapeString(obj.Comment.Text),
				Rating:      obj.Rating,
				Source_date: obj.Source_date,
				Photos:      photo,
				Posted_at:   int64(posted_at.Unix()),
				Scraped_at:  int64(time.Now().Unix()),
			}

			for _, obj := range obj.OwnerReply {
				response := OwnerReply{
					Author_name: obj.Author_name.Name,
					Text:        html.UnescapeString(obj.Text),
					Posted_at:   obj.Source_date,
				}
				review.OwnerReply = append(review.OwnerReply, response)
			}

			for _, preObj := range obj.PreviousReview {
				posted_at, err := time.Parse("1/2/2006", preObj.Source_date)
				checkError(err)

				var photo []string
				for _, photoObj := range preObj.Photos {
					photo = append(photo, photoObj.Src)
				}

				previous := ReviewFomate{
					Parent_id:   obj.Review_id,
					Review_id:   preObj.Review_id,
					Author_id:   preObj.Author_id,
					Author_name: preObj.User.Author_name,
					Text:        html.UnescapeString(preObj.Comment.Text),
					Rating:      preObj.Rating,
					Source_date: preObj.Source_date,
					Photos:      photo,
					Posted_at:   int64(posted_at.Unix()),
					Scraped_at:  int64(time.Now().Unix()),
				}

				for _, obj := range preObj.OwnerReply {
					response := OwnerReply{
						Author_name: obj.Author_name.Name,
						Text:        html.UnescapeString(obj.Text),
						Posted_at:   obj.Source_date,
					}
					previous.OwnerReply = append(previous.OwnerReply, response)
				}

				safeReviewAdd(previous)
			}

			safeReviewAdd(review)
			// reviews = append(reviews, review)
			rev_counter += 1
		}
		fmt.Println("Count", (rev_counter + non_counter))
		wg.Done() // done REVIEW call [success]
	})
	return linkCall
}

func nonRecommandedReviewUrlCall(spider *Spider, wg *sync.WaitGroup, link string) {
	linkCall := getColly(spider.Persona.Proxy)
	linkCall.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
		wg.Done() // done NON_RECOMMENDED_ONCE call [failed]
	})
	linkCall.OnHTML(`html`, func(e *colly.HTMLElement) {
		fmt.Println("Response - ", e.Request.URL.String())

		nonReviewCount := 0

		for _, v := range e.ChildTexts("h3") {
			if strings.Contains(v, "recommended") {
				re := regexp.MustCompile("(\\d+)")
				match := re.FindStringSubmatch(v)
				if len(match) >= 2 {
					count, err := strconv.Atoi(match[1])
					if err != nil {
						panic(err)
					}
					nonReviewCount = count
					if count == 0 {
						wg.Done() // done NON_RECOMMENDED_ONCE call [success - without reviews]
						fmt.Println("No review")
						scrapStatus = "NO_REVIEWS"
						return
					}
				}
			}
		}

		fmt.Println("Non recommanded Reviews", nonReviewCount)
		minimal_review_count = nonReviewCount

		nonRecommandedCollector := nonRecommandedReviewUrlCallFollowup(spider, wg)
		for i := 0; i < nonReviewCount; i += 10 {
			wg.Add(1) // add NON_RECOMMENDED_REV call
			visitingUrl := e.Request.URL.String() + "?not_recommended_start=" + strconv.Itoa(i)
			nonRecommandedCollector.Visit(visitingUrl)
		}
		wg.Done() // done NON_RECOMMENDED_ONCE call [success]
	})
	linkCall.Visit(link)
}

func nonRecommandedReviewUrlCallFollowup(spider *Spider, wg *sync.WaitGroup) *colly.Collector {
	linkCall := getColly(spider.Persona.Proxy)
	linkCall.OnError(func(r *colly.Response, e error) {
		log.Println("error:", e, r.Request.URL, string(r.Body))
		wg.Done() // done NON_RECOMMENDED_REV call [failed]
	})
	linkCall.OnHTML(`html`, func(e *colly.HTMLElement) {
		nonReviewCount := len(e.ChildTexts(`div.not-recommended-reviews > ul.reviews > li`))
		wg.Add(nonReviewCount) // add NON_REV_COUNT call
		wg.Done()              // done NON_RECOMMENDED_REV call [sucecss]
		fmt.Println("Count", (rev_counter + non_counter))
	})
	linkCall.OnHTML(`div.not-recommended-reviews > ul.reviews > li`, func(e *colly.HTMLElement) {
		var author_id string
		author_id_string := e.ChildAttr("div.review-sidebar .user-display-name", "href")
		if author_id_string != "" {
			autherRe := regexp.MustCompile(`'userid=(.*)`)
			author_id = autherRe.FindStringSubmatch(author_id_string)[0]
		}

		author_name := e.ChildText("div.review-sidebar .user-display-name")
		text := e.ChildText("div.review-wrapper div.review-content p")

		date := strings.Fields(e.ChildText("div.review-wrapper div.review-content .rating-qualifier"))
		source_date := date[0]

		review_id := e.ChildAttr("div.review--with-sidebar", "data-review-id")

		re := regexp.MustCompile(`regular-\s*(\d+)`)
		rat := re.FindStringSubmatch(e.ChildAttr(".biz-rating .i-stars", "class"))[1]
		rating, _ := strconv.Atoi(rat)

		var photos []string
		photo := e.ChildAttr("ul.photo-box-grid div.photo-box img.photo-box-img", "data-async-src")
		if photo != "" {
			photos = append(photos, photo)
		}

		posted_at, err := time.Parse("1/2/2006", source_date)
		checkError(err)

		review := ReviewFomate{
			Review_id:       review_id,
			Author_id:       author_id,
			Author_name:     author_name,
			Text:            html.UnescapeString(text),
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
			response := OwnerReply{
				Author_name: strings.Replace(e.ChildText("div.biz-owner-reply-header strong"), "Comment from ", "", -1),
				Text:        html.UnescapeString(e.ChildText("span.js-content-toggleable.hidden")),
				Posted_at:   source_date,
			}

			review.OwnerReply = append(review.OwnerReply, response)
		}

		// if review has previous review
		var previousReviewText string
		previousReviewText = e.ChildText("div.review-wrapper div.previous-review span.js-expandable-comment span.js-content-toggleable")

		if previousReviewText != "" {
			date := strings.Fields(e.ChildText("div.review-wrapper div.previous-review .rating-qualifier"))
			source_date := date[0]
			posted_at, err := time.Parse("1/2/2006", source_date)
			checkError(err)

			rat := re.FindStringSubmatch(e.ChildAttr(".previous-review .biz-rating .i-stars", "class"))[1]
			rating, _ := strconv.Atoi(rat)

			var photos []string
			photo := e.ChildText(".previous-review ul.photo-box-grid div.photo-box img.photo-box-img")
			if photo != "" {
				photos = append(photos, photo)
			}

			previous := ReviewFomate{
				Parent_id:       review_id,
				Author_id:       author_id,
				Author_name:     author_name,
				Text:            html.UnescapeString(previousReviewText),
				Rating:          rating,
				Source_date:     source_date,
				Not_recommended: true,
				Photos:          photos,
				Posted_at:       int64(posted_at.Unix()),
				Scraped_at:      int64(time.Now().Unix()),
			}
			safeReviewAdd(previous)
		}

		safeReviewAdd(review)
		// reviews = append(reviews, review)
		non_counter += 1
		wg.Done() // done NON_REV_COUNT call [success]
	})
	return linkCall
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

func dumpReviews(fname string) {
	for _, v := range reviews {
		_, err := WriteDataToFileAsJSON(v, fname)
		if err != nil {
			panic(err)
		}
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

func dumpMetaData(spider *Spider) {
	data := Meta{
		Histogram:          histogram,
		Profile_key:        spider.ProfileKey,
		Item_scraped_count: item_scraped_count,
		Scraping_status:    scrapStatus,
		Start_time:         start_time,
		Finish_time:        finish_time,
		Request_count:      requestCount,
		Response_bytes:     responseBytes,
	}
	mainFileExt := filepath.Ext(spider.filename)
	fnameIndex := len(spider.filename) - len(mainFileExt)
	metaFile := spider.filename[0:fnameIndex] + "-meta.json"
	WriteDataToFileAsJSON(data, metaFile)
}

func safeReviewAdd(review ReviewFomate) {
	mu.Lock()
	applyHashKey(&review)
	encodeFielsToB64(&review)
	dt, _ := time.Parse("1/2/2006", review.Source_date)
	i := 0
	for ; i < len(reviews); i++ {
		rdt, _ := time.Parse("1/2/2006", reviews[i].Source_date)
		
		if rdt.Before(dt) {
			break;
		}
	}
	if (len(reviews) > 0  && i < len(reviews)) {
		last := len(reviews) - 1
		reviews = append(reviews, reviews[last])
		copy(reviews[i+1:], reviews[i:last])	 
		reviews[i] = review
	} 
	if len(reviews) == i {
		reviews = append(reviews, review)
	}
	mu.Unlock()
}

func applyHashKey(review *ReviewFomate) {
	// First prepare string to make Hash
	lstForHash := []string{}

	x := review
	if !hasText(x) && !hasAuthor(x) && !hasResponses(x) && hasRevId(x) {
		// no text, no author, no responses but id exists
		lstForHash = append(lstForHash, x.Review_id)
	} else if hasResponses(x) {
		// responses exists and it's first response has text
		lstForHash = append(lstForHash, x.Text)
		lstForHash = append(lstForHash, x.Author_name)
		lstForHash = append(lstForHash, x.OwnerReply[0].Text)
	} else {
		// use text and author for generating hash
		lstForHash = append(lstForHash, review.Text)
		lstForHash = append(lstForHash, review.Author_name)
	}
	rawStr, _ := json.Marshal(lstForHash)
	h := md5.New()
	io.WriteString(h, string(rawStr))
	review.ReviewHash = hex.EncodeToString(h.Sum(nil))
}

func hasText(review *ReviewFomate) bool {
	return review.Text != ""
}

func hasAuthor(review *ReviewFomate) bool {
	return review.Author_name != ""
}

func hasResponses(r *ReviewFomate) bool {
	return len(r.OwnerReply) > 0 && r.OwnerReply[0].Text != ""
}

func hasRevId(review *ReviewFomate) bool {
	return review.Review_id != ""
}

func encodeFielsToB64(review *ReviewFomate) {
	if hasText(review) {
		review.Text = base64.StdEncoding.EncodeToString([]byte(review.Text))
	}
	if hasAuthor(review) {
		review.Author_name = base64.StdEncoding.EncodeToString([]byte(review.Author_name))
	}
	if hasResponses(review) {
		for key, obj := range review.OwnerReply {
			review.OwnerReply[key].Text = base64.StdEncoding.EncodeToString([]byte(obj.Text))
			review.OwnerReply[key].Author_name = base64.StdEncoding.EncodeToString([]byte(obj.Author_name))
		}
	}

}
