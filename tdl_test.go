package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// bundle gives a directory that looks like an app bundle's MacOS folder, where
// the program itself sits under a name that matches tdl on a filesystem that
// ignores case.
func bundle(t *testing.T) (self string, contents string) {
	t.Helper()
	root := t.TempDir()
	contents = filepath.Join(root, "TDL.app", "Contents")
	macos := filepath.Join(contents, "MacOS")
	if err := os.MkdirAll(macos, 0o755); err != nil {
		t.Fatal(err)
	}
	self = filepath.Join(macos, "TDL")
	if err := os.WriteFile(self, []byte("program"), 0o755); err != nil {
		t.Fatal(err)
	}
	return self, contents
}

func TestFindBinarySkipsTheRunningProgram(t *testing.T) {
	self, _ := bundle(t)
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	got, err := findBinary(config{}, self)
	if err == nil && sameFile(got, self) {
		t.Fatalf("findBinary returned the program itself：%q", got)
	}
}

func TestFindBinaryTakesTheResourcesCopy(t *testing.T) {
	self, contents := bundle(t)
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	want := filepath.Join(contents, "Resources", "tdl")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("准备的文件不在：%v", err)
	}
	got, err := findBinary(config{}, self)
	if err != nil {
		t.Fatal(err)
	}
	if !sameFile(got, want) {
		t.Fatalf("findBinary 返回 %q，期望 %q", got, want)
	}
}

func TestFindBinaryExpandsTheHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	want := filepath.Join(home, "bin", "tdl")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findBinary(config{Binary: "~/bin/tdl"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !sameFile(got, want) {
		t.Fatalf("findBinary 返回 %q，期望 %q", got, want)
	}
}

func TestFindBinaryReadsTheSetting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	want := filepath.Join(dir, "tdl-custom")
	if err := os.WriteFile(want, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findBinary(config{Binary: want}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("findBinary 返回 %q，期望 %q", got, want)
	}
}

// TestFindBinaryReadsTheGoBinDirectory finds tdl where go install puts it, which
// the short PATH of an app opened from the Dock leaves out.
func TestFindBinaryReadsTheGoBinDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())

	want := filepath.Join(home, "go", "bin", "tdl")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findBinary(config{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("findBinary 返回 %q，期望 %q", got, want)
	}
}

// script writes an executable shell script and returns its path.
func script(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("这段测试用 shell 脚本模拟 tdl")
	}
	path := filepath.Join(t.TempDir(), "tdl")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestProbeVersionReadsTheBanner takes the version a real tdl prints.
func TestProbeVersionReadsTheBanner(t *testing.T) {
	bin := script(t, `echo "Version: 0.20.4"
echo "Commit: 9d7d49e"`)
	got, err := probeVersion(bin)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.20.4" {
		t.Fatalf("probeVersion 返回 %q，期望 0.20.4", got)
	}
}

// TestProbeVersionRejectsASilentFile keeps a file that is not tdl out of the
// setting, so a download never runs it.
func TestProbeVersionRejectsASilentFile(t *testing.T) {
	bin := script(t, `exit 3`)
	if _, err := probeVersion(bin); err == nil {
		t.Fatal("一个不打印版本的文件被当成了 tdl")
	}
}
