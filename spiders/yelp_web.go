package spiders

import (
    "go-yelp-with-proxy/services"
)

type YelpSpider struct {
  BaseSpider
  allowedDomains []string
}

func (me *YelpSpider) AddDomain(domain string) {
  me.allowedDomains = append(me.allowedDomains, domain)
}

func (me *YelpSpider) Setup() services.Scrapy {
  scrapy := services.Scrapy{}
  me.AddDomain("yelp.com")
  me.AddDomain("www.yelp.com")
  
  scrapy.Setup(me.allowedDomains)

  scrapy.SetProxy(me.Persona.Proxy)
  return scrapy
}

func (me *YelpSpider) Run() {
  scrapy := me.Setup()
  scrapy.Request(me.ProfileKey)
}
