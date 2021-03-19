package spiders

import (
	// "fmt"
	"encoding/base64"
	"strings"
	"encoding/json"
)

type BaseSpider struct {
	ProfileKey       string   `json:"profile_key"`
	BusinessName     string   `json:"business_name"`
	LastReviewHashes []string `json:"last_review_hashes"`
	BusinessID       int      `json:"business_id"`
	ClientID         int      `json:"client_id"`
	BatchID          int      `json:"batch_id"`
	TaskID           int      `json:"task_id"`
	Persona struct{
		AdditionalCookies interface{} `json:"additional_cookies"`
		Proxy             string      `json:"proxy"`
		OtherProxies      []string    `json:"other_proxies"`
	} `json:"persona"`
	Address struct{
		City   string `json:"city"`
		State  string `json:"state"`
		Street string `json:"street"`
		Zip    string `json:"zip"`
	} `json:"address"`
	filename string
}

func (me *BaseSpider) SetAdditionalArgs(arg string) {
	additionalArgs := strings.Split(arg, "=")
	_, p := additionalArgs[0], additionalArgs[1]
  place, err := base64.StdEncoding.DecodeString(p)
  if err != nil {
  	panic(err)
  }
	err = json.Unmarshal(place, &me)
  if err != nil {
  	panic(err)
  }
}

func (me *BaseSpider) SetOutputFilename(fname string) {
	me.filename = fname
}