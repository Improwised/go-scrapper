package config

import (
	"go-yelp-with-proxy/spiders"
)

var Spiders = map[string]spiders.RootSpider {
  "yelp-web": &spiders.YelpSpider{},
}
