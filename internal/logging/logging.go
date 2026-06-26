// Package logging configures the application logger. It mirrors the Python
// rich-based logger (logger.py): a single named, colored, timestamped logger.
package logging

import (
	"os"

	"github.com/charmbracelet/log"
)

// Log is the package-level logger, analogous to Python's
// logging.getLogger("seline-agent").
var Log *log.Logger

func init() {
	Log = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Prefix:          "seline-agent",
	})
	Log.SetLevel(log.InfoLevel)
}
