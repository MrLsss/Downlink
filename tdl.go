package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

// exeName is the name of the tdl binary on this platform.
func exeName() string {
	if runtime.GOOS == "windows" {
		return "tdl.exe"
	}
	return "tdl"
}

// probeEnv marks a process that this program started to ask a binary for its
// version. The program itself exits on it instead of opening a window.
const probeEnv = "DOWNLINK_PROBE"

// findBinary returns the tdl binary the settings name, else one of the usual
// places it ships in. self is the program that runs this code: its own
// directory holds it under a name tdl matches on a filesystem that ignores
// case, and running it in place of tdl would start this program again.
func findBinary(cfg config, self string) (string, error) {
	var candidates []string
	if p := strings.TrimSpace(cfg.Binary); p != "" {
		candidates = append(candidates, expandHome(p))
	}
	if p, err := exec.LookPath(exeName()); err == nil {
		candidates = append(candidates, p)
	}
	// An app started from the Dock gets a short PATH, so look where the usual
	// ways of installing tdl put it.
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "go", "bin", exeName()),
			filepath.Join(home, ".local", "bin", exeName()),
			filepath.Join(home, ".tdl", "bin", exeName()))
	}
	candidates = append(candidates,
		filepath.Join("/opt/homebrew/bin", exeName()),
		filepath.Join("/usr/local/bin", exeName()))
	if dir, err := mygo.App.Path(mygo.PathResources); err == nil {
		candidates = append(candidates, filepath.Join(dir, exeName()), filepath.Join(dir, "bin", exeName()))
	}
	if self != "" {
		dir := filepath.Dir(self)
		candidates = append(candidates,
			filepath.Join(dir, exeName()),
			filepath.Join(dir, "..", "Resources", exeName()))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "tdl_MacOS_arm64", exeName()),
			filepath.Join(wd, "tdl_MacOS_amd64", exeName()),
			filepath.Join(wd, "tdl_Windows_amd64", exeName()),
			filepath.Join(wd, "tdl_Windows_arm64", exeName()),
			filepath.Join(wd, "tdl_Linux_amd64", exeName()),
			filepath.Join(wd, "tdl_Linux_arm64", exeName()),
			filepath.Join(wd, exeName()))
	}
	for _, p := range candidates {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}
		if self != "" && sameFile(p, self) {
			continue
		}
		return p, nil
	}
	return "", errors.New("找不到 tdl 可执行文件，请在设置里指定它的位置")
}

// sameFile reports whether two paths name the same file.
func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

// probeVersion runs tdl version and returns what it reports.
func probeVersion(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "version")
	cmd.Env = append(os.Environ(), "NO_COLOR=1", probeEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		out = append([]byte(err.Error()+" "), out...)
		return "", fmt.Errorf("无法运行 %s：%s", path, strings.TrimSpace(string(out)))
	}
	v := parseVersion(string(out))
	if v == "" {
		return "", fmt.Errorf("%s 不是 tdl：%s", path, strings.TrimSpace(string(out)))
	}
	return v, nil
}

// parseVersion reads the version out of tdl version output.
func parseVersion(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "Version: "); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// globalArgs are the flags every tdl command takes.
func (a *app) globalArgs() []string {
	args := []string{
		"-n", a.cfg.Namespace,
		"-t", itoa(int64(a.cfg.Threads)),
		"-l", itoa(int64(a.cfg.Limit)),
	}
	if p := strings.TrimSpace(a.cfg.Proxy); p != "" {
		args = append(args, "--proxy", p)
	}
	return args
}

// downloadArgs builds the download command line.
func (a *app) downloadArgs(urls []string, dir string) []string {
	args := a.globalArgs()
	args = append(args, "dl")
	for _, u := range urls {
		args = append(args, "-u", u)
	}
	args = append(args, "-d", dir)
	if s := strings.TrimSpace(a.cfg.Include); s != "" {
		args = append(args, "-i", s)
	}
	if s := strings.TrimSpace(a.cfg.Exclude); s != "" {
		args = append(args, "-e", s)
	}
	if s := strings.TrimSpace(a.cfg.Template); s != "" {
		args = append(args, "--template", s)
	}
	if a.cfg.SkipSame {
		args = append(args, "--skip-same")
	}
	if a.cfg.RewriteExt {
		args = append(args, "--rewrite-ext")
	}
	if a.cfg.Group {
		args = append(args, "--group")
	}
	if a.cfg.Takeout {
		args = append(args, "--takeout")
	}
	if a.cfg.Restart {
		args = append(args, "--restart")
	} else {
		args = append(args, "--continue")
	}
	return args
}

// loginArgs builds the login command line.
func (a *app) loginArgs() []string {
	return append(a.globalArgs(), "login", "-T", "code")
}

// links splits what the user pasted into links.
func links(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
