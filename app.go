package main

import (
	_ "embed"
	"flag"
	"fmt"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
)

func main() {
	var addr *string
	var path = "/mnt"
	var uname *string
	var upass *string
	var errs int64
	addr = flag.String("addr", ":80", "")
	uname = flag.String("uname", "zxc", "")
	upass = flag.String("upass", "zxc", "")
	flag.Parse()
	fmt.Println(*addr, path, *uname, *upass)
	fss := &webdav.Handler{
		FileSystem: webdav.Dir(path),
		LockSystem: webdav.NewMemLS(),
	}
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if errs > 5 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "WebDAV: login time is more", http.StatusUnauthorized)
			return
		}
		if req.Method == http.MethodDelete {
			w.WriteHeader(webdav.StatusLocked)
			w.Write([]byte("can't delete"))
			return
		}
		username, password, ok := req.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			errs += 1
			return
		}
		if username != *uname || password != *upass {
			http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
			return
		}

		if req.Method == "GET" {
			http.FileServer(http.Dir(path)).ServeHTTP(w, req)
			return
		}

		fss.ServeHTTP(w, req)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Panicln(err)
	}
}
