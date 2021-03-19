package main

import (
    "github.com/spf13/cobra"
    "go-yelp-with-proxy/cli"
    "go-yelp-with-proxy/logger"
    "go-yelp-with-proxy/settings"
)

func main() {
  Init()
}

func Init() {
  cfg := settings.GetAppSettings()
  logger.InitLogger(cfg.Debug, cfg.IsDevelopment)

  /* Root Command ! */
  var rootCmd = &cobra.Command{
    Use:   "crawl",
  }
  cli.RegisterSpiders(rootCmd)
  if err := rootCmd.Execute(); err != nil {
    panic(err)
  }
}