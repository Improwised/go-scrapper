package spiders

import (
    "fmt"
)

type YelpSpider struct {
  BaseSpider
}

func (b *YelpSpider) Setup() {
  fmt.Println("Runnnig spider from base.")
}

func (b *YelpSpider) Run() {
  fmt.Println("Runnnig spider from base.")
}
