package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func homeDir() (string, error) { return os.UserHomeDir() }

// tdlDataDir is where tdl keeps its sessions and logs.
func tdlDataDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		log.Printf("找不到用户目录：%v", err)
		return ""
	}
	return filepath.Join(h, ".tdl")
}

// namespaces lists the sessions tdl keeps in its data directory, one file per
// namespace. The single file older versions wrote, data.kv, is left out: tdl
// forks off it only while the data directory does not exist, and once it does,
// that file holds no session tdl reads.
func namespaces() []string {
	root := tdlDataDir()
	out := []string{}
	if entries, err := os.ReadDir(filepath.Join(root, "data")); err == nil {
		for _, e := range entries {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// hasSession reports whether tdl wrote a session for the namespace.
func hasSession(ns string) bool {
	if ns == "" {
		return false
	}
	for _, n := range namespaces() {
		if n == ns {
			return true
		}
	}
	return false
}

// openPath shows a file or a folder in the desktop's file manager.
func openPath(path string) {
	if path == "" {
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("打开 %s 失败：%v", path, err)
	}
}

// revealPath shows where a file is, with the file itself selected.
func revealPath(path string) {
	if path == "" {
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,"+path)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}
	if err := cmd.Start(); err != nil {
		log.Printf("显示 %s 失败：%v", path, err)
	}
}
