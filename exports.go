package loki

import "github.com/edaniel30/loki-logger-go/models"

// Re-export the models
type Config = models.Config
type Option = models.Option
type Level = models.Level
type Fields = models.Fields

// Re-export the default config
// Returns a Config with sensible defaults
var DefaultConfig = models.DefaultConfig
