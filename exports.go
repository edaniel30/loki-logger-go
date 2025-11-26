package loki

import "github.com/edaniel30/loki-logger-go/models"

// Re-export types
type Config = models.Config
type Option = models.Option
type Level = models.Level
type Fields = models.Fields
type Entry = models.Entry

// Re-export level constants
const (
	LevelDebug = models.LevelDebug
	LevelInfo  = models.LevelInfo
	LevelWarn  = models.LevelWarn
	LevelError = models.LevelError
	LevelFatal = models.LevelFatal
)

// Re-export functions
var (
	DefaultConfig = models.DefaultConfig
	NewEntry      = models.NewEntry
	ParseLevel    = models.ParseLevel
)

// Re-export configuration options
var (
	WithAppName       = models.WithAppName
	WithLokiHost      = models.WithLokiHost
	WithLokiUsername  = models.WithLokiUsername
	WithLokiPassword  = models.WithLokiPassword
	WithLogLevel      = models.WithLogLevel
	WithLabels        = models.WithLabels
	WithBatchSize     = models.WithBatchSize
	WithFlushInterval = models.WithFlushInterval
	WithMaxRetries    = models.WithMaxRetries
	WithTimeout       = models.WithTimeout
	WithOnlyConsole   = models.WithOnlyConsole
)
