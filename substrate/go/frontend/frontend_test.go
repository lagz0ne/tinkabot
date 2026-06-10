package frontend

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

var assetRef = regexp.MustCompile(`/(assets/[^"]+)`)

func TestEmbeddedFrontendIndex(t *testing.T) {
	t.Parallel()
	body, err := Index()
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)
	if !strings.Contains(html, `<div id="app">`) {
		t.Fatalf("embedded index missing app root: %s", html)
	}
	if !strings.Contains(html, "/assets/") {
		t.Fatalf("embedded index missing built asset ref: %s", html)
	}
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range assetRef.FindAllStringSubmatch(html, -1) {
		if _, err := fs.Stat(files, match[1]); err != nil {
			t.Fatalf("embedded index references missing asset %q: %v", match[1], err)
		}
	}
}

func TestEmbeddedFrontendFiles(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(files, "index.html"); err != nil {
		t.Fatal(err)
	}
}
