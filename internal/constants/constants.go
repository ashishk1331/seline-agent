// Package constants holds runtime constants ported from constants.py.
package constants

import "github.com/ashishk1331/seline-agent/internal/config"

// TinyFishHeaders returns the headers used for TinyFish search/fetch requests.
func TinyFishHeaders(cfg *config.Config) map[string]string {
	return map[string]string{
		"X-API-Key":    cfg.TinyfishAPIKey,
		"Content-Type": "application/json",
	}
}

// ThinkingPhrases are the Hinglish "thinking" status-bubble messages.
var ThinkingPhrases = []string{
	"soch rahi hoon...",
	"ek second...",
	"haan haan, dekh rahi hoon...",
	"abhi batati hoon...",
	"hmm...",
	"thoda socha jaaye...",
	"ek minute...",
	"haan, samjhi...",
	"dekhte hain...",
}
