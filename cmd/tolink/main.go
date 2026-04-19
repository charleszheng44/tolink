package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/charleszheng44/tolink/pkg/server"
	"github.com/charleszheng44/tolink/pkg/store"
	"github.com/charleszheng44/tolink/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4080", "Address to listen on (e.g. 127.0.0.1:4080 or :80)")
	defaultData := filepath.Join(os.Getenv("HOME"), ".config", "tolink", "links.json")
	data := flag.String("data", defaultData, "Path to links JSON file")
	flag.Parse()

	s, err := store.New(*data)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	srv := server.New(s, web.Content)
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, srv))
}
