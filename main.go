package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func parseArgs() []string {
	args := os.Args[1:]
	var positional []string
	for len(args) > 0 {
		flag.CommandLine.Parse(args)
		args = flag.Args()
		if len(args) > 0 {
			positional = append(positional, args[0])
			args = args[1:]
		}
	}
	return positional
}

func main() {
	group := flag.Bool("group", false, "group by process")
	flag.BoolVar(group, "g", false, "shorthand for --group")
	port := flag.Int("port", 0, "filter by port")
	flag.IntVar(port, "p", 0, "shorthand for --port")
	force := flag.Bool("force", false, "kill with SIGKILL")
	flag.BoolVar(force, "f", false, "shorthand for --force")
	showVersion := flag.Bool("version", false, "print version")
	positional := parseArgs()
	if *showVersion {
		fmt.Println("kp", version)
		return
	}

	ls, err := listListeners()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if len(positional) > 0 {
		os.Exit(runKill(os.Stdout, os.Stderr, newKiller(), ls, positional, *force))
	}

	if *port != 0 {
		ls = filterByPort(ls, *port)
	}
	sortByPort(ls)

	if len(ls) == 0 {
		fmt.Println("no listeners found")
		return
	}

	if *group {
		printGrouped(os.Stdout, groupByProcess(ls))
		return
	}
	printTable(os.Stdout, ls)
}
