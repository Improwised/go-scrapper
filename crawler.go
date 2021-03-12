package main

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "go-yelp-with-proxy/cli"
)

func main() {
  Init()
}


func Init() {
  var rootCmd = &cobra.Command{
    Use:   "crawl",
  }
  // rootCmd.AddCommand(cli.GetSpiderCmdDef("yelp"))
  cli.RegisterSpiders(rootCmd)
  if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}