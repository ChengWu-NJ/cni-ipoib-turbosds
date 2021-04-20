package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	version = "1.0.0"
	commit  = "will be replaced when make"
	date    = "will be replaced when make"
)

func main() {
	// Init command line flags to clear vendor packages' flags, especially in init()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// add version flag
	versionOpt := false
	flag.BoolVar(&versionOpt, "version", false, "Show application version")
	flag.BoolVar(&versionOpt, "v", false, "Show application version")
	flag.Parse()
	if versionOpt {
		fmt.Printf("cni-ipoib-ts version:%s, commit:%s, date:%s\n", version, commit, date)
		return
	}

	//TODO....
	//invoke pkg/runbashscript and return result to stdout

}
