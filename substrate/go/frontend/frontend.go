package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:site
var site embed.FS

func Files() (fs.FS, error) {
	return fs.Sub(site, "site")
}

func Index() ([]byte, error) {
	return site.ReadFile("site/index.html")
}
