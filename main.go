package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	data := flag.String("data", "nodes.json", "node data file")
	admin := flag.String("admin", "", "admin token for the management panel (required)")
	flag.Parse()
	if *admin == "" {
		// Allow env var as an alternative to the CLI flag.
		*admin = os.Getenv("SUBPANEL_ADMIN")
	}
	if *admin == "" {
		fmt.Fprintln(os.Stderr, "error: set -admin TOKEN or SUBPANEL_ADMIN environment variable")
		os.Exit(2)
	}
	log.Printf("subpanel listening on %s (data=%s)", *addr, *data)
	if err := serve(*addr, *data, *admin); err != nil {
		log.Fatal(err)
	}
}
