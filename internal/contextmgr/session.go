// Package contextmgr manages the conversation context and its on-disk session.
// Go port of the context/ package (session.py + __init__.py).
package contextmgr

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ashishk1331/seline-agent/internal/config"
	"github.com/ashishk1331/seline-agent/internal/types"
	petname "github.com/dustinkirkland/golang-petname"
)

// Consumption is the token-usage snapshot persisted alongside a session.
// JSON keys match the Python implementation for file compatibility.
type Consumption struct {
	CurrentTokens          int     `json:"current_tokens"`
	CurrentTokensInWords   string  `json:"current_tokens_in_words"`
	MaxTokens              int     `json:"max_tokens"`
	MaxTokensInWords       string  `json:"max_tokens_in_words"`
	RemainingTokens        int     `json:"remaining_tokens"`
	RemainingTokensInWords string  `json:"remaining_tokens_in_words"`
	PercentageUsed         float64 `json:"percentage_used"`
}

// Session handles persistence of messages and consumption to disk.
type Session struct {
	cfg       *config.Config
	SessionID string
}

// NewSession resolves (or generates) a session ID and initializes its files.
func NewSession(cfg *config.Config, sessionID string) (*Session, error) {
	s := &Session{cfg: cfg}
	if sessionID != "" {
		s.SessionID = sessionID
	} else {
		s.SessionID = s.findLastSessionID()
	}
	if err := s.initialize(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Session) sessionsDir() string {
	return filepath.Join(s.cfg.WorkspaceDir, "sessions")
}

func (s *Session) sessionFilePath() string {
	return filepath.Join(s.sessionsDir(), s.SessionID+".jsonl")
}

func (s *Session) lastSessionIDPath() string {
	return filepath.Join(s.sessionsDir(), "last_session_id.txt")
}

func (s *Session) consumptionFilePath() string {
	return filepath.Join(s.sessionsDir(), s.SessionID+"_consumption.json")
}

func (s *Session) initialize() error {
	if err := os.MkdirAll(s.sessionsDir(), 0o755); err != nil {
		return err
	}
	if err := touch(s.sessionFilePath()); err != nil {
		return err
	}
	if err := touch(s.consumptionFilePath()); err != nil {
		return err
	}
	return os.WriteFile(s.lastSessionIDPath(), []byte(s.SessionID), 0o644)
}

func (s *Session) findLastSessionID() string {
	if b, err := os.ReadFile(s.lastSessionIDPath()); err == nil {
		if id := strings.TrimSpace(string(b)); id != "" {
			return id
		}
	}
	return petname.Generate(4, "-")
}

func (s *Session) appendMessageToSession(msg types.Message) error {
	f, err := os.OpenFile(s.sessionFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

func (s *Session) overwriteMessagesInSession(messages []types.Message) error {
	f, err := os.Create(s.sessionFilePath())
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, m := range messages {
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return w.Flush()
}

func (s *Session) loadMessagesFromSession() ([]types.Message, error) {
	f, err := os.Open(s.sessionFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var messages []types.Message
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m types.Message
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, sc.Err()
}

func (s *Session) dumpConsumption(c Consumption) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(s.consumptionFilePath(), b, 0o644)
}

func (s *Session) retrieveConsumption() Consumption {
	var c Consumption
	b, err := os.ReadFile(s.consumptionFilePath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c) // empty/invalid -> zero value
	return c
}

func touch(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
