package main

import (
  "encoding/json"
  "strconv"
  "fmt"
  "log"
  "os"
  "github.com/gocolly/colly/v2"
)


func main() {
  fName := "resultm.json"
  file, err := os.Create(fName)
  if err != nil {
    log.Fatalf("Cannot create file %q: %s\n", fName, err)
    return
  }
  defer file.Close()
  var result map[string]interface{}
  var cout = 0
  var stopFlag = true
  // Instantiate default collector
  // export $HTTP_PROXY = "65c0f90ccf854cb5874088f30da2d82c:@odmarkj.crawlera.com:8010"
  c := colly.NewCollector(
    // Visit only domains: coursera.org, www.coursera.org
    colly.AllowedDomains("yelp.com", "www.yelp.com"),

    // Cache responses to prevent multiple download of pages
    // even if the collector is restarted
    // colly.CacheDir("./coursera_cache"),
  )
  // fixedURL, err:= url.Parse("65c0f90ccf854cb5874088f30da2d82c@odmarkj.crawlera.com:8010")
  // if err != nil {
  // panic(err)
  // }
  // c.SetProxyFunc(http.ProxyURL(fixedURL))
  // c.WithTransport(&http.Transport{
  //   Proxy: http.ProxyURL(fixedURL),
  //   DialContext: (&net.Dialer{
  //     Timeout:   30 * time.Second,
  //     KeepAlive: 30 * time.Second,
  //     DualStack: true,
  //   }).DialContext,
  //   MaxIdleConns:          100,
  //   IdleConnTimeout:       90 * time.Second,
  //   TLSHandshakeTimeout:   10 * time.Second,
  //   ExpectContinueTimeout: 1 * time.Second,
  // })
  // Create another collector
  detailCollector := c.Clone()

  c.OnRequest(func(r *colly.Request) {
        fmt.Println("Visiting", r.URL)
    })

    c.OnHTML("meta", func(e *colly.HTMLElement) {
        // id = e.Attr("content")
    if e.Attr("name") == "yelp-biz-id"{
      stopFlag = true
      for stopFlag {
        detailCollector.Visit("https://www.yelp.com/biz/nR2dFrY7VnYzJ1gtdkA5mw/review_feed?rl=en&sort_by=relevance_desc&q=&start="+ strconv.Itoa(cout))
        cout = cout + 20
      }
    }
    })

    c.OnScraped(func(r *colly.Response) { 
        // data, err := json.Marshal(result)
        // if err != nil {
        //     fmt.Println(err)
        // } else {
        //     fmt.Println("Finished. Here is your data:", string(data))
        // }
    })

  // Before making a request print "Visiting ..."
  c.OnRequest(func(r *colly.Request) {
    log.Println("visiting", r.URL.String())
  })

  // Extract details
  detailCollector.OnResponse(func(res *colly.Response) {
    err := json.Unmarshal(res.Body, &result) 
    fmt.Printf("result: %+v", result["pagination"])
    fmt.Printf("rrr: %+v", err)
  })

  // Start scraping
  c.Visit("https://www.yelp.com/biz/home-alarm-authorized-adt-dealer-lemon-grove")

  enc := json.NewEncoder(file)
  enc.SetIndent("", "  ")

  // Dump json to the standard output
  enc.Encode(result)
}