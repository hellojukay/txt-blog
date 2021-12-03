package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var bind = flag.String("bind", "127.0.0.1:8000", "port to run the server on")
var dir = flag.String("dir", ".", "server root directory")

func main() {
	flag.Parse()

	httpdir := http.Dir(*dir)
	handler := renderer{httpdir, http.FileServer(httpdir)}

	log.Println("Serving on http://" + *bind)
	log.Fatal(http.ListenAndServe(*bind, handler))
}

// 匹配 markdown 中的图片 ![img](src/img/big.png)
var re = regexp.MustCompile(`!.*?\((.*?)\)`)

type renderer struct {
	d http.Dir
	h http.Handler
}

func (r renderer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.URL.Path, ".txt") && !strings.HasSuffix(req.URL.Path, ".md") {
		r.h.ServeHTTP(rw, req)
		return
	}
	var pathErr *os.PathError
	input, err := ioutil.ReadFile(string(r.d) + req.URL.Path)
	if errors.As(err, &pathErr) {
		http.Error(rw, http.StatusText(http.StatusNotFound)+": "+req.URL.Path, http.StatusNotFound)
		log.Printf("file not found: %s", err)
		return
	}

	if err != nil {
		http.Error(rw, "Internal Server Error: "+err.Error(), 500)
		log.Printf("Couldn't read path %s: %v (%T)", req.URL.Path, err, err)
		return
	}
	// 展示图片
	var output = re.ReplaceAllString(string(input), `</pre><img src="$1" style="max-width:100%%;max-height: 600px;" /><pre>`)

	output = fmt.Sprintf("<html><body><pre>%s</pre></html></body>", output)
	fmt.Fprintf(rw, output)
}
