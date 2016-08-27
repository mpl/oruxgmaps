// Copyright 2016 Mathieu "mpl" Lonjaret

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	idstring = "http://golang.org/pkg/http/#ListenAndServe"
)

var (
	host = flag.String("host", "0.0.0.0:8080", "listening port and hostname")
	help = flag.Bool("h", false, "show this help")
)

var uploadTmpl *template.Template

func usage() {
	fmt.Fprintf(os.Stderr, "\t simpleHttpd \n")
	flag.PrintDefaults()
	os.Exit(2)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				http.Error(w, e.Error(), http.StatusInternalServerError)
				return
			}
		}()
		title := r.URL.Path
		w.Header().Set("Server", idstring)
		fn(w, r, title)
	}
}

func uploadHandler(rw http.ResponseWriter, req *http.Request, url string) {
	if req.Method == "GET" {
		if err := uploadTmpl.Execute(rw, nil); err != nil {
			log.Printf("template error: %v", err)
		}
		return
	}

	mr, err := req.MultipartReader()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var data []byte
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(rw, "reading body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		fileName := part.FileName()
		if fileName == "" {
			continue
		}
		buf := bytes.NewBuffer(make([]byte, 0))
		lr := io.LimitedReader{
			R: part,
			N: 1 << 20,
		}
		_, err = io.Copy(buf, &lr)
		if err != nil {
			http.Error(rw, "copying: "+err.Error(), http.StatusInternalServerError)
			return
		}
		data, err = insert(buf)
		if err != nil {
			log.Printf("error inserting gmaps def: %v", err)
			http.Error(rw, "error inserting gmaps def", http.StatusInternalServerError)
			return
		}
		break
	}
	if _, err := io.Copy(rw, bytes.NewReader(data)); err != nil {
		log.Printf("error serving onlinemapsources.xml: %v", err)
		return
	}
}

var uploadHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>Upload files</title>
</head>
<body>
  <h1>Upload files</h1>

  <form action="/upload" method="POST" id="uploadform" enctype="multipart/form-data">
    <input type="file" id="fileinput" multiple="false" name="file">
    <input type="submit" id="filesubmit" value="Upload">
  </form>

</body>
</html>
`

func main() {
	flag.Usage = usage
	flag.Parse()

	uploadTmpl = template.Must(template.New("upload").Parse(uploadHTML))
	http.HandleFunc("/upload", makeHandler(uploadHandler))
	fmt.Println("Starting to listen on: https://" + *host)
	if err := http.ListenAndServeTLS(
		*host,
		filepath.Join(os.Getenv("HOME"), "keys", "cert.pem"),
		filepath.Join(os.Getenv("HOME"), "keys", "key.pem"),
		nil); err != nil {
		log.Fatal(err)
	}
}

func insert(r io.Reader) ([]byte, error) {
	lr := io.LimitedReader{
		R: r,
		N: 1 << 20,
	}

	var (
		dataBefore []byte
		dataAfter  []byte
	)

	// Parsing XML with regexp is bad, mkay?
	endOfMapSourcesRxp := regexp.MustCompile("^</onlinemapsources>")
	mapSourceRxp := regexp.MustCompile(`(\s)*<onlinemapsource uid="([0-9]+)">.*`)

	lastuid := 0
	isAfter := false
	sc := bufio.NewScanner(&lr)
	for sc.Scan() {
		l := sc.Bytes()
		if endOfMapSourcesRxp.Match(l) {
			isAfter = true
			dataAfter = append(dataAfter, append(l, '\n')...)
			break
		}
		l = append(l, '\n')
		if isAfter {
			dataAfter = append(dataAfter, l...)
		} else {
			dataBefore = append(dataBefore, l...)
		}
		m := mapSourceRxp.FindSubmatch(l)
		if len(m) == 0 {
			continue
		}
		uid, err := strconv.Atoi(string(m[2]))
		if err != nil {
			return nil, err
		}
		lastuid = uid
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	lastuid++
	gmapsDef := fmt.Sprintf(
		`
<onlinemapsource uid="%d">
<name>Google Maps</name>
<url><![CDATA[http://mt{$s}.google.com/vt/lyrs=m@121&hl={$l}&x={$x}&y={$y}&z={$z}]]></url>
<website><![CDATA[<a href="http://url.to.website">web site link</a>]]></website>
<minzoom>0</minzoom>
<maxzoom>19</maxzoom>
<projection>MERCATORESFERICA</projection>
<servers>0,1,2,3</servers>
<httpparam name=""></httpparam>
<cacheable>1</cacheable>
<downloadable>1</downloadable>
<maxtilesday>0</maxtilesday>
<maxthreads>0</maxthreads>
<xop></xop>
<yop></yop>
<zop></zop>
<qop></qop>
<sop></sop>
</onlinemapsource>
`, lastuid)

	return append(append(dataBefore, []byte(gmapsDef)...), dataAfter...), nil
}
