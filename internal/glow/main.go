package glow

import (
	"fmt"
	"os"
)

type Glow struct {
	Version string
}

func New() *Glow {
	return &Glow{
		Version: "0.1.0",
	}
}

func (g *Glow) CmdRun() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsageAndExit()
	}
	if args[0] == "version" {
		printVersionAndExit(g)
	}
	if args[0] == "help" {
		printUsageAndExit()
	}
}

func printUsageAndExit() {
	fmt.Println("Usage: glow [command]")
	os.Exit(1)
}

func printVersionAndExit(g *Glow) {
	fmt.Println(g.Version)
	os.Exit(0)
}
