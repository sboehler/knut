package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strconv"
)

//go:embed build
var assets embed.FS

func Files() (http.Handler, error) {
	var f fs.FS
	if ok, _ := strconv.ParseBool(os.Getenv("KNUT_WEB_DIRECT")); ok {
		f = os.DirFS("web/build")
	} else {
		rooted, err := fs.Sub(assets, "build")
		if err != nil {
			return nil, err
		}
		f = rooted
	}
	return http.FileServer(http.FS(f)), nil

}
