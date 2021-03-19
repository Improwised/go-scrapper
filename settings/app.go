package settings

import (
  "github.com/joho/godotenv"
  "github.com/kelseyhightower/envconfig"
)

// AppConfig type AppConfig
type AppConfig struct {
  IsDevelopment bool   `envconfig:"IS_DEVELOPMENT"`
  Debug         bool   `envconfig:"DEBUG"`
  Env           string `envconfig:"APP_ENV"`
}

// GetConfig Collects all configs
func GetAppSettings() AppConfig {
  _ = godotenv.Load()

  AllConfig := AppConfig{}

  err := envconfig.Process("", &AllConfig)
  if err != nil {
    panic(err)
  }

  return AllConfig
}