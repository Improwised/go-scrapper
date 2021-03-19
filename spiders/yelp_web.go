package spiders

import (
    "go-yelp-with-proxy/services"
    "go-yelp-with-proxy/logger"
)

type YelpSpider struct {
  BaseSpider
  allowedDomains []string
}

func (me *YelpSpider) AddDomain(domain string) {
  me.allowedDomains = append(me.allowedDomains, domain)
}

func (me *YelpSpider) Setup() services.Scrapy {
  me.AddDomain("yelp.com")
  me.AddDomain("www.yelp.com")

  scrapy := services.Scrapy{}
  scrapy.Setup(me.allowedDomains)
  scrapy.SetProxy(me.Persona.Proxy)
  return scrapy
}


func (me *YelpSpider) ParseBusinessPage() {
  log := logger.GetLogger()
  log.Info("I'm here at parse business page !")
}

func (me *YelpSpider) Run() {

  /* Init Logger */
  log := logger.GetLogger()

  scrapy := me.Setup()
  // headers := map[string]string {
  //   "X-Crawlera-Profile": "desktop",  
  // }

  log.Info("Starting profile request")
  // scrapy.Request(me.ProfileKey, me.ParseBusinessPage, headers)
  log.Info("ProfileKey = " + me.ProfileKey)
  scrapy.CheckMe(me.ProfileKey)
}
