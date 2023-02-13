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
	"os"
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

var (
	addr *string
	path = "/mnt"
	//path     = "/home/dong"
	uname    *string
	upass    *string
	errs     int64
	errMAx   *int64
	isDocker *int64
)

func main() {
	log.SetFlags(log.Llongfile)
	fmt.Println("v1.9.6")
	addr = flag.String("addr", ":80", "")
	uname = flag.String("uname", "zxc", "")
	upass = flag.String("upass", "zxc", "")
	errMAx = flag.Int64("maxerr", 0, "")
	isDocker = flag.Int64("docker", 1, "")
	flag.Parse()

	StartDb()

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

		if req.Method == "DELETE" {
			if idf&DeleteMode != DeleteMode {
				http.Error(w, "WebDAV: access defined!", http.StatusForbidden)
				return
			}
		}
		if req.Method == "PUT" || req.Method == "MKCOL" {
			if idf&WriteMode != WriteMode {
				http.Error(w, "WebDAV: access defined!", http.StatusForbidden)
				return
			}
		}
		if req.Method == "MOVE" || req.Method == "PROPPATCH" {
			if idf&UpdateMode != UpdateMode {
				http.Error(w, "WebDAV: access defined!", http.StatusForbidden)
				return
			}
		}
		if req.Method == "GET" {
			if idf&ReadMode == ReadMode {
				truePath := path
				if sharePath != "" {
					truePath += sharePath + req.URL.Path
				} else {
					truePath += req.URL.Path
				}
				info, err := os.Stat(truePath)
				if err == nil && info != nil && info.IsDir() == false {
					SendFile(w, req, truePath)
					return
				}
				tmp, err := template.New("index").Parse(indexHtml)
				if err != nil {
					log.Println(err)
				}

				data := GetFileInDir(truePath)
				for _, i2 := range data.Files {
					i2.Path = "/" + strings.TrimPrefix(req.URL.Path+"/"+i2.Name, "//")
					if strings.HasPrefix(i2.Path, "//") {
						i2.Path = strings.TrimPrefix(i2.Path, "/")
					}
					i2.Share = shareName
				}
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

		goto end
	end:
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
	Path  string
	Share string
}
type FileWalk struct {
	Total int // 总共大小
	Files []*FileSingle
}

// GetFileInDir 获取目录下的文件
func GetFileInDir(path string) *FileWalk {
	cmd, err := exec.
		Command("sh", "-c", fmt.Sprintf(`ls '%s' -lA -h --no-group -g --time-style=+%s | awk '{print($1,$3,$1="",$2="",$3="",$4="",$0)}'`, path, "%H")).
		CombinedOutput()
	if err != nil {
		//log.Println(err)
	}
	// 开始解析
	buf := bufio.NewReader(bytes.NewReader(cmd))
	filewalk := &FileWalk{}
	for {
		line, _, c := buf.ReadLine()
		if c == io.EOF {
			break
		}
		if bytes.HasPrefix(line, []byte("total")) {
			filewalk.Total, err = strconv.Atoi(string(line[6:]))
			continue
		}
		// 直接省略前3个字段
		//counter := 0
		f := &FileSingle{
			IsDir: false,
			Size:  "",
			Name:  "",
		}
		first := 11
		if line[0] == 'd' {
			f.IsDir = true
		}
		end := 11
		for i := 11; i < len(line); i++ {
			// 获取到size
			if f.Size == "" && line[i] != ' ' {
				first = i
				end = i
				for line[end] != ' ' {
					end += 1
				}
				i = end
				f.Size = string(line[first:end])
				continue
			}
			// 获取到name
			if f.Name == "" && line[i] != ' ' {
				first = i
				f.Name = string(line[first:])
				continue
			}
		}
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

func SendFile(writer http.ResponseWriter, request *http.Request, path string) {
	f, _ := os.Open(path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		log.Println("sendFile1", err.Error())
		http.NotFound(writer, request)
		return
	}
	writer.Header().Add("Accept-Ranges", "bytes")
	writer.Header().Add("Content-Disposition", "attachment; filename="+info.Name())
	var start, end int64
	//fmt.Println(request.Header,"\n")
	if r := request.Header.Get("Range"); r != "" {
		if strings.Contains(r, "bytes=") && strings.Contains(r, "-") {

			fmt.Sscanf(r, "bytes=%d-%d", &start, &end)
			if end == 0 {
				end = info.Size() - 1
			}
			if start > end || start < 0 || end < 0 || end >= info.Size() {
				writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				log.Println("sendFile2 start:", start, "end:", end, "size:", info.Size())
				return
			}
			writer.Header().Add("Content-Length", strconv.FormatInt(end-start+1, 10))
			writer.Header().Add("Content-Range", fmt.Sprintf("bytes %v-%v/%v", start, end, info.Size()))
			writer.WriteHeader(http.StatusPartialContent)
		} else {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		writer.Header().Add("Content-Length", strconv.FormatInt(info.Size(), 10))
		start = 0
		end = info.Size() - 1
	}
	_, err = f.Seek(start, 0)
	if err != nil {
		log.Println("sendFile3", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	n := 1024
	buf := make([]byte, n)
	for {
		if end-start+1 < int64(n) {
			n = int(end - start + 1)
		}
		_, err := f.Read(buf[:n])
		if err != nil {
			log.Println("1:", err)
			if err != io.EOF {
				log.Println("error:", err)
			}
			return
		}
		err = nil
		_, err = writer.Write(buf[:n])
		if err != nil {
			//log.Println(err, start, end, info.Size(), n)
			return
		}
		start += int64(n)
		if start >= end+1 {
			return
		}
	}

}
