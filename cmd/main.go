// Copyright 2016 Mathieu "mpl" Lonjaret

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mpl/oruxgmaps"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: oruxgmaps /path/to/onlinemapsources.xml > output.xml")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()

	f, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	lr := &io.LimitedReader{
		R: f,
		N: 1 << 20,
	}

	withGmaps, err := oruxgmaps.Insert(lr)
	if err != nil {
		log.Fatalf("error inserting gmaps definition: %v", err)
	}
	fmt.Printf("%s\n", withGmaps)
}
