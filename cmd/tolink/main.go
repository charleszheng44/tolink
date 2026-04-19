package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/charleszheng44/tolink/pkg/server"
	"github.com/charleszheng44/tolink/pkg/store"
	"github.com/charleszheng44/tolink/web"
)

func main() {
	port := flag.Int("port", 80, "Port to listen on")
	defaultData := filepath.Join(os.Getenv("HOME"), ".config", "tolink", "links.json")
	data := flag.String("data", defaultData, "Path to links JSON file")
	flag.Parse()

	s, err := store.New(*data)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	srv := server.New(s, web.Content)
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
