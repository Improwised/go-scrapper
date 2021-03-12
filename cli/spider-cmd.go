package cli

import (
    "fmt"
    "github.com/spf13/cobra"
    "go-yelp-with-proxy/config"
)

func GetSpiderCmdDef(SpiderName string) *cobra.Command {
  var cmd = &cobra.Command{
    Use:   SpiderName,
    Short: "Run spider " + SpiderName,
    Long: "Run spider " + SpiderName,
    Run: func(cmd *cobra.Command, args []string) {
      // Do Stuff Here
      fmt.Println("Running Spider " + SpiderName)
      spider := config.Spiders[SpiderName]
      additional_args := cmd.Flag("additional-args").Value.String()
      spider.SetAdditionalArgs(additional_args)
      spider.Run()
    },
  }
  cmd.PersistentFlags().StringP("additional-args", "a", "", "NAME=VALUE as additional Arguments.")
  return cmd
}

func RegisterSpiders(rootCmd *cobra.Command) {
  for SpiderName, _ := range config.Spiders {
    SpiderCommand := GetSpiderCmdDef(SpiderName)
    rootCmd.AddCommand(SpiderCommand)
  }
}