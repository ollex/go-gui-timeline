// Command webserver serves a built timeline WASM example on loopback.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	dir := flag.String("dir", "build/timeline-web", "directory to serve")
	flag.Parse()

	info, err := os.Stat(*dir)
	if err != nil {
		log.Fatalf("web root: %v", err)
	}
	if !info.IsDir() {
		log.Fatalf("web root %q is not a directory", *dir)
	}

	files := http.FileServer(http.Dir(*dir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
	url := "http://" + *addr + "/"
	fmt.Printf("Timeline WebAssembly test is available at %s\n", url)
	fmt.Println("Press Ctrl+C to stop the server.")
	log.Fatal(http.ListenAndServe(*addr, handler))
}
