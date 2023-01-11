package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"golang.org/x/net/webdav"
	"html/template"
	"io"
	"log"
	"net/http"
	"os/exec"
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
	Path     string // 限定那个链接
	PathPass string // 分享的密码
}

var users = make(map[string]Access)

var (
	//go:embed index.html
	indexHtml string
)

func main() {
	log.SetFlags(log.Llongfile)
	var addr *string
	var path = "/mnt"
	var uname *string
	var upass *string
	var errs int64
	var errMAx *int64
	addr = flag.String("addr", ":80", "")
	uname = flag.String("uname", "zxc", "")
	upass = flag.String("upass", "zxc", "")
	errMAx = flag.Int64("maxerr", 0, "")
	flag.Parse()

	fmt.Println(*addr, path, *uname, *upass, *errMAx)
	fss := &webdav.Handler{
		FileSystem: webdav.Dir(path),
		LockSystem: webdav.NewMemLS(),
	}
	users[*uname] = Access{
		Pass:     *upass,
		Identify: ReadMode | WriteMode | DeleteMode | UpdateMode,
	}
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// TODO 接下来是关于分享的修改
		shareName := req.URL.Query().Get("share")
		sharePass := req.URL.Query().Get("pass")
		// 权限
		idf := 0
		sharePath := ""
		if shareName != "" {
			share := GetShare(shareName)
			if share.Path == "" {
				w.WriteHeader(404)
				w.Write([]byte("文件不支持或者取消分享了"))
				return
			}
			if sharePass != share.Pass {
				w.Write([]byte("校验密码错误"))
				return
			}
			idf = share.Idf
			sharePath = share.Path
		}
		if *errMAx != 0 && errs > *errMAx {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "WebDAV: login time is more", http.StatusUnauthorized)
			return
		}
		username, password, ok := req.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
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
				tmp, err := template.New("index").Parse(indexHtml)
				if err != nil {
					log.Println(err)
				}
				truePath := path
				if sharePath != "" {
					truePath += sharePath
				} else {
					truePath += req.URL.Path
				}
				data := GetFileInDir(truePath)
				err = tmp.Execute(w, data)
				if err != nil {
					log.Println(err)
				}
				return
				//http.FileServer(http.Dir(path)).ServeHTTP(w, req)
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

	// 应该有一个分享的功能

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Panicln(err)
	}
}

type FileSingle struct {
	IsDir bool   // 是否是目录
	Size  string // 大小
	Name  string // 名称
}
type FileWalk struct {
	Total int // 总共大小
	Files []*FileSingle
}

// GetFileInDir 获取目录下的文件
func GetFileInDir(path string) *FileWalk {
	cmd, err := exec.Command("ls", path, "-lA", "--no-group", "-g").CombinedOutput()
	if err != nil {
		log.Println(err)
	}
	// 开始解析
	buf := bufio.NewReader(bytes.NewReader(cmd))
	filewalk := &FileWalk{}
	for {
		line, _, c := buf.ReadLine()
		if c == io.EOF {
			break
		}
		str := string(line)
		if strings.HasPrefix(str, "total") {
			filewalk.Total, err = strconv.Atoi(str[6:])
			continue
		}
		// 直接省略前3个字段
		counter := 0
		f := &FileSingle{
			IsDir: false,
			Size:  "",
			Name:  "",
		}
		if str[0] == 'd' {
			f.IsDir = true
		}
		first := 0
		for i := 0; i < len(str)-1; i++ {
			if str[i] == ' ' {
				counter += 1
				if counter <= 2 {
					first = i
					continue
				}
				key := line[first:i]
				if len(key) == 1 && key[0] == 32 {
					first = i
					continue
				}
				if f.Size == "" {
					n, _ := strconv.Atoi(str[first+1 : i])
					f.Size = HumanFileSize(n)
				}
				first = i
			}
		}
		f.Name = str[first+1:]
		filewalk.Files = append(filewalk.Files, f)
	}
	return filewalk
}
func HumanFileSize(size int) string {
	if size >= 1<<30 {
		return strconv.Itoa(size/(1<<30)) + "GB"
	}
	if size >= 1<<20 {
		return strconv.Itoa(size/(1<<20)) + "MB"
	}
	if size >= 1<<10 {
		return strconv.Itoa(size/(1<<10)) + "KB"
	}
	return strconv.Itoa(size) + " bytes"
}
