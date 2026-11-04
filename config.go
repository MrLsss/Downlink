package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
)

const (
	statusDone      = "done"
	statusFailed    = "failed"
	statusCancelled = "cancelled"
)

// account is the Telegram account the last successful login reported.
type account struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func (a account) label() string {
	if a.Username != "" {
		return "@" + a.Username
	}
	if a.ID != 0 {
		return "ID " + itoa(a.ID)
	}
	return ""
}

// record is one finished download, kept in the history.
type record struct {
	ID     string    `json:"id"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Dir    string    `json:"dir"`
	URLs   []string  `json:"urls"`
	Files  int       `json:"files"`
	Bytes  int64     `json:"bytes"`
	Status string    `json:"status"`
	Error  string    `json:"error"`
}

// config is what the app remembers between runs.
type config struct {
	Binary     string   `json:"binary"`
	Namespace  string   `json:"namespace"`
	Proxy      string   `json:"proxy"`
	Dir        string   `json:"dir"`
	Threads    int      `json:"threads"`
	Limit      int      `json:"limit"`
	SkipSame   bool     `json:"skipSame"`
	RewriteExt bool     `json:"rewriteExt"`
	Group      bool     `json:"group"`
	Takeout    bool     `json:"takeout"`
	Restart    bool     `json:"restart"`
	Include    string   `json:"include"`
	Exclude    string   `json:"exclude"`
	Template   string   `json:"template"`
	Account    account  `json:"account"`
	History    []record `json:"history"`
}

// dataDir is where the app keeps its settings, overridable for tests.
func dataDir() string {
	if d := os.Getenv("TDL_UI_DATA_DIR"); d != "" {
		return d
	}
	d, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		log.Fatal(err)
	}
	return d
}

func configPath() string { return filepath.Join(dataDir(), "config.json") }

// legacyConfigPath is where the app kept its settings under its former name.
func legacyConfigPath() string {
	return filepath.Join(filepath.Dir(dataDir()), "TDL", "config.json")
}

func downloadDir() string {
	if d, err := mygo.App.Path(mygo.PathDownloads); err == nil {
		return filepath.Join(d, "tdl")
	}
	return "tdl-downloads"
}

// defaultProxy is the proxy the settings start with; Telegram's own servers are
// reachable from this machine only through it.
const defaultProxy = "http://127.0.0.1:7890"

func defaultConfig() config {
	return config{
		Namespace: "default",
		Proxy:     defaultProxy,
		Dir:       downloadDir(),
		Threads:   4,
		Limit:     2,
	}
}

func loadConfig() (config, error) {
	cfg := defaultConfig()
	moved := false
	b, err := os.ReadFile(configPath())
	if os.IsNotExist(err) {
		// The app used to be called TDL; a settings file left under that name
		// is the user's own, so it comes along.
		b, err = os.ReadFile(legacyConfigPath())
		if err != nil {
			return cfg, nil
		}
		moved = true
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.fix()
	if moved {
		if err := saveConfig(cfg); err != nil {
			log.Printf("旧设置没有搬过来：%v", err)
		}
	}
	return cfg, nil
}

// fix repairs the values a settings file may lack.
func (c *config) fix() {
	d := defaultConfig()
	if c.Namespace == "" {
		c.Namespace = d.Namespace
	}
	if c.Dir == "" {
		c.Dir = d.Dir
	}
	if c.Threads < 1 {
		c.Threads = d.Threads
	}
	if c.Limit < 1 {
		c.Limit = d.Limit
	}
}

func saveConfig(cfg config) error {
	if err := os.MkdirAll(dataDir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, configPath())
}

const historyLimit = 40

// pushHistory adds a finished download to the top of the history.
func (c *config) pushHistory(rec record) {
	c.History = append([]record{rec}, c.History...)
	if len(c.History) > historyLimit {
		c.History = c.History[:historyLimit]
	}
}

func (c *config) removeHistory(id string) {
	out := c.History[:0]
	for _, r := range c.History {
		if r.ID != id {
			out = append(out, r)
		}
	}
	c.History = out
}
