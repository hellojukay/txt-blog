package main

import (
	_ "embed"
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

//go:embed custom.js
var script string

//go:embed custom.css
var style string
var bind = flag.String("bind", "127.0.0.1:8000", "port to run the server on")
var dir = flag.String("dir", ".", "server root directory")

func main() {
	flag.Parse()

	httpdir := http.Dir(*dir)
	handler := renderer{httpdir, http.FileServer(httpdir)}

	log.Println("Serving on http://" + *bind)
	log.Fatal(http.ListenAndServe(*bind, handler))
}

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
	var title = get_title(string(input))
	var head = fmt.Sprintf(`
<title>%s</title>
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.3.1/styles/gml.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.3.1/highlight.min.js"></script>
<style>%s</style>
`, title, style)
	var output = render_code(string(input))
	// 展示图片

	output = render_img(output)

	output = fmt.Sprintf("<html><head>%s</head><body><div><pre>%s</pre></div></body><script>%s</script></html>", head, output, script)
	fmt.Fprint(rw, output)
}

// 渲染代码块
func render_code(html string) string {

	var re = regexp.MustCompile("(?s)```([a-z]+)\r?\n(.*?)\r?\n```")
	html = re.ReplaceAllString(html, `</pre><pre><code class="language-$1">$2</code></pre><pre>`)
	re = regexp.MustCompile("(?s)```\r?\n(.*?)\r?\n```")
	return re.ReplaceAllString(html, `</pre><pre><code class="language-plaintext">$1</code></pre><pre>`)
}

func render_link(html string) string {
	// 匹配 markdown 中的图片 ![img](src/img/big.png)
	var re = regexp.MustCompile(`\[.*?\]\((.*?)\)`)
	return re.ReplaceAllString(html, `</pre><a href="$1"/>$1</a><pre>`)
}
func render_img(html string) string {
	// 匹配 markdown 中的图片 ![img](src/img/big.png)
	var re = regexp.MustCompile(`!.*?\((.*?)\)`)
	return re.ReplaceAllString(html, `</pre><img src="$1"/><pre>`)

}
func get_title(html string) string {
	var re = regexp.MustCompile(`title:\s?(.*?)\r?\n`)
	var res = re.FindAllStringSubmatch(html, -1)
	if len(res) > 0 {
		return res[0][1]
	}
	return ""
}
