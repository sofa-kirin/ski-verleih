package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"time"

	"skiverleih/internal/api"
	"skiverleih/internal/store"
)

//go:embed static
var static embed.FS

func main() {
	addr := flag.String("addr", ":8080", "Adresse des HTTP-Servers")
	ohneSeed := flag.Bool("leer", false, "ohne Beispieldaten starten")
	flag.Parse()

	st := store.New()
	if !*ohneSeed {
		st.Seed(time.Now())
	}

	mux := http.NewServeMux()
	api.New(st, time.Now).Routes(mux)
	statisch, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /", http.FileServerFS(statisch))

	log.Printf("Ski-Verleih läuft auf http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
