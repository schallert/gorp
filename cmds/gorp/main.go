package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/schallert/gorp"
)

var (
	listenAddr = flag.String("addr", "localhost:8080", "address to listen for http requests on")
	rAddr      = flag.String("raddr", "localhost:6311", "address of rserve daemon (must be on localhost)")
)

func main() {
	flag.Parse()

	fmt.Printf("Listening on %s\n", *listenAddr)

	_, err := gorp.NewServer(*listenAddr, *rAddr)
	if err != nil {
		log.Fatalln(err)
	}

	select {}
}

func chk(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
