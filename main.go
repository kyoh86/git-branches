package main

import (
	"log"
	"os"

	"github.com/alecthomas/kingpin"
)

func main() {
	var (
		directory string
		color     bool

		filters []string
		force   bool
	)

	app := kingpin.New("git-branches", "Manage branches with interfaces")

	listCmd := app.Command("list", "Show each branch, upstream, author in git repository").Default()
	listCmd.Flag("directory", "Run as if git was started in <path> instead of the current working directory.").Short('C').StringVar(&directory)
	listCmd.Flag("color", "Output with ANSI colors").BoolVar(&color)
	listCmd.Arg("filter", "Filter").EnumsVar(&filters, filterNames...)

	cleanupCmd := app.Command("cleanup", "Cleanup dead (lost upstream) branch")
	cleanupCmd.Flag("directory", "Run as if git was started in <path> instead of the current working directory.").Short('C').StringVar(&directory)
	cleanupCmd.Flag("force", "Delete dead branches force. If it is on, calls `git branch -D` instead of `git branch -d`").BoolVar(&force)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case listCmd.FullCommand():
		if err := list(directory, color, filters); err != nil {
			log.Fatal(err)
		}
	case cleanupCmd.FullCommand():
		if err := cleanup(force); err != nil {
			log.Fatal(err)
		}
	}
}
