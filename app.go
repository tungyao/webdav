package main

import (
	_ "embed"
	"flag"
	"fmt"
	"golang.org/x/net/webdav"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

const (
	RunModeServe  = 0x11
	RunModeClient = 0x12
)

const (
	ReadMode = 0x1 << iota
	WriteMode
	DeleteMode
	UpdateMode
)

type Access struct {
	Pass     string
	Identify int
}

var users = make(map[string]Access)

func main() {
	var addr *string
	var path = "/mnt"
	var uname *string
	var upass *string
	var errs int64
	var errMAx *int64
	var runMode uint8
	addr = flag.String("addr", ":7000", "")
	uname = flag.String("uname", "zxc", "")
	upass = flag.String("upass", "zxc", "")
	errMAx = flag.Int64("maxerr", 0, "")
	flag.Parse()
	arg := flag.Args()
	// 确定运行方式
	for _, cm := range arg {
		if cm == "client" {
			runMode = RunModeClient
			break
		}
	}
	if runMode == RunModeClient {
		AccountPanelClient()
	}

	fmt.Println(*addr, path, *uname, *upass, *errMAx)
	fss := &webdav.Handler{
		FileSystem: webdav.Dir(path),
		LockSystem: webdav.NewMemLS(),
	}
	users[*uname] = Access{
		Pass:     *upass,
		Identify: ReadMode | WriteMode | DeleteMode | UpdateMode,
	}
	go AccountPanel()
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Println(req.Method)
		if *errMAx != 0 && errs > *errMAx {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "WebDAV: login time is more", http.StatusUnauthorized)
			return
		}
		username, password, ok := req.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			errs += 1
			return
		}
		idf := 0
		for k, v := range users {
			if k == username && v.Pass == password {
				idf = v.Identify
				break
			}
		}
		if idf == 0 {
			http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
		}

		if req.Method == "GET" {
			if idf&ReadMode == ReadMode {
				http.FileServer(http.Dir(path)).ServeHTTP(w, req)
			} else {
				http.Error(w, "WebDAV: access defined!", http.StatusForbidden)
			}
			return
		}
		if req.Method == "DELETE" {
			if idf&DeleteMode != DeleteMode {
				http.Error(w, "WebDAV: access defined!", http.StatusForbidden)
				return
			}
		}
		if req.Method == "UPDATE" {
			if idf&UpdateMode != UpdateMode {
				http.Error(w, "WebDAV: access defined!", http.StatusForbidden)
				return
			}
		}
		if req.Method == "POST" {
			if idf&WriteMode != WriteMode {
				http.Error(w, "WebDAV: access defined!", http.StatusForbidden)
				return
			}
		}
		fss.ServeHTTP(w, req)
	})

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Panicln(err)
	}
}

func AccountPanelClient() {

	conn, err := net.Dial("unix", "/home/dong/webdav.sock")
	if err != nil {
		log.Fatalln(err)
	}
	var cmd string
	for {
		fmt.Print("cmd:\n")
		_, err = fmt.Scanf("%s", &cmd)
		if err != nil {
			fmt.Print("\nerr ", err.Error(), "\ncmd\n")
			return
		}
		if cmd == "exit" || cmd == "quit" {
			conn.Write([]byte("exit"))
			conn.Close()
			os.Exit(0)
			return
		}
		log.Println(cmd)
		_, err := conn.Write([]byte(cmd))
		if err != nil {
			fmt.Print("\nerr ", err.Error(), "\ncmd\n")
			return
		}

	}
}

func AccountPanel() {
	ux, err := net.Listen("unix", "/home/dong/webdav.sock")
	if err != nil {
		log.Fatalln(err)
	}
	go func() {
		if err := recover(); err != nil {
			log.Println(err)
			os.Remove("/home/dong/webdav.sock")
		}
	}()
	defer os.Remove("/home/dong/webdav.sock")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-c:
			os.Remove("/home/dong/webdav.sock")
			os.Exit(0)
		}
	}()
	for {
		conn, err := ux.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go func() {
			defer conn.Close()
			for {
				readData := make([]byte, 128)
				n, _ := conn.Read(readData)
				if readData[0] == 0 {
					return
				}
				data := strings.Split(string(readData[:n]), ",")
				switch data[0] {
				case "add":
					if len(data) == 4 {
						idf, err := strconv.Atoi(data[3])
						log.Println(idf, err)
						users[data[1]] = Access{
							Pass:     data[2],
							Identify: idf,
						}
					}
				case "del":
					delete(users, data[1])
				case "upd":
					if len(data) == 4 {
						idf, _ := strconv.Atoi(data[3])
						users[data[1]] = Access{
							Pass:     data[2],
							Identify: idf,
						}
					}
				case "exit":
					return
				}
				log.Println(users)
				conn.Write([]byte("ok"))
			}

		}()

	}

}
