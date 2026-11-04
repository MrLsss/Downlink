package main

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeTDL writes the layout tdl leaves in its data directory.
func fakeTDL(t *testing.T, namespaces ...string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".tdl", "data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, ns := range namespaces {
		if err := os.WriteFile(filepath.Join(dir, ns), []byte("bolt"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func TestNamespacesReadsTheDataDirectory(t *testing.T) {
	fakeTDL(t, "alpha", "beta")
	got := namespaces()
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("namespaces() = %v", got)
	}
	if !hasSession("beta") {
		t.Fatal("hasSession(\"beta\") = false")
	}
	if hasSession("gamma") {
		t.Fatal("hasSession(\"gamma\") = true")
	}
}

// TestNamespacesSkipsTheOldSingleFile leaves out data.kv, which tdl forks off
// only while its data directory does not exist.
func TestNamespacesSkipsTheOldSingleFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".tdl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".tdl", "data.kv"), []byte("kv"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := namespaces(); len(got) != 0 {
		t.Fatalf("namespaces() = %v", got)
	}
}

func TestFirstRunTakesTheOnlySession(t *testing.T) {
	fakeTDL(t, "mrlsss")
	t.Setenv("TDL_UI_DATA_DIR", filepath.Join(t.TempDir(), "app"))

	a := newApp()
	a.load()
	if a.cfg.Namespace != "mrlsss" {
		t.Fatalf("首次启动的命名空间是 %q", a.cfg.Namespace)
	}
}

// TestTheOnlySessionReplacesANamespaceWithoutOne covers the settings an earlier
// version left naming a namespace tdl has no session for.
func TestTheOnlySessionReplacesANamespaceWithoutOne(t *testing.T) {
	fakeTDL(t, "mrlsss")
	dir := filepath.Join(t.TempDir(), "app")
	t.Setenv("TDL_UI_DATA_DIR", dir)

	cfg := defaultConfig()
	cfg.Namespace = "default"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	a := newApp()
	a.load()
	if a.cfg.Namespace != "mrlsss" {
		t.Fatalf("命名空间变成了 %q", a.cfg.Namespace)
	}
}

func TestLaterRunsKeepTheChosenNamespace(t *testing.T) {
	fakeTDL(t, "mrlsss", "work")
	dir := filepath.Join(t.TempDir(), "app")
	t.Setenv("TDL_UI_DATA_DIR", dir)

	cfg := defaultConfig()
	cfg.Namespace = "work"
	if err := saveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	a := newApp()
	a.load()
	if a.cfg.Namespace != "work" {
		t.Fatalf("已有设置文件时命名空间变成了 %q", a.cfg.Namespace)
	}
}

func TestProxyIsSetForNewSettings(t *testing.T) {
	fakeTDL(t)
	t.Setenv("TDL_UI_DATA_DIR", filepath.Join(t.TempDir(), "app"))
	a := newApp()
	a.load()
	if a.cfg.Proxy != defaultProxy {
		t.Fatalf("新的设置里代理是 %q", a.cfg.Proxy)
	}
}
