// Copyright 2016 Mathieu "mpl" Lonjaret

package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mpl/oruxgmaps"
)

const (
	idstring = "http://golang.org/pkg/http/#ListenAndServe"
)

var (
	host = flag.String("host", "0.0.0.0:4430", "listening port and hostname")
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

func rootHandler(w http.ResponseWriter, r *http.Request, url string) {
	if r.Method != "GET" {
		http.Error(w, "not a GET", http.StatusMethodNotAllowed)
		return
	}
	http.Redirect(w, r, "/upload", http.StatusFound)
}

func uploadHandler(w http.ResponseWriter, r *http.Request, url string) {
	if r.Method == "GET" {
		if err := uploadTmpl.Execute(w, nil); err != nil {
			log.Printf("template error: %v", err)
		}
		return
	}

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var data []byte
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "reading body: "+err.Error(), http.StatusInternalServerError)
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
			http.Error(w, "copying: "+err.Error(), http.StatusInternalServerError)
			return
		}
		data, err = oruxgmaps.Insert(buf)
		if err != nil {
			log.Printf("error inserting gmaps def: %v", err)
			http.Error(w, "error inserting gmaps def", http.StatusInternalServerError)
			return
		}
		break
	}

	h := w.Header()
	h.Set("Content-Type", "application/octet-stream")
	h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "onlinemapsources.xml"}))
	w.WriteHeader(http.StatusOK)
	http.ServeContent(w, r, "onlinemapsources.xml", time.Now(), bytes.NewReader(data))
}

// TODO(mpl): make it a one click action, like in camli. but needs js afair.
var uploadHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>Upload</title>
</head>
<body>
  <h1>Upload your onlinemapsources.xml</h1>

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
	http.HandleFunc("/", makeHandler(rootHandler))
	fmt.Println("Starting to listen on: https://" + *host)
	if err := http.ListenAndServeTLS(
		*host,
		filepath.Join(os.Getenv("HOME"), "keys", "cert.pem"),
		filepath.Join(os.Getenv("HOME"), "keys", "key.pem"),
		nil); err != nil {
		log.Fatal(err)
	}
}

