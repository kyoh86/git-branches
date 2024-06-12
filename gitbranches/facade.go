package gitbranches

import (
	"log"
	"os"

	"github.com/alecthomas/kingpin"
)

func Run(app *kingpin.Application) {
	var (
		directory string
		color     bool

		filters []string
		force   bool
	)

	app.Flag("directory", "Run as if git was started in <path> instead of the current working directory.").Short('C').StringVar(&directory)

	listCmd := app.Command("list", "Show each branch, upstream, author in git repository").Default()
	listCmd.Flag("color", "Output with ANSI colors").Default("true").BoolVar(&color)
	listCmd.Arg("filter", "Filter").EnumsVar(&filters, filterNames...)

	cleanupCmd := app.Command("cleanup", "Cleanup dead (lost upstream) branch")
	cleanupCmd.Flag("force", "Delete dead branches force. If it is on, calls `git branch -D` instead of `git branch -d`").BoolVar(&force)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case listCmd.FullCommand():
		if err := list(directory, color, filters); err != nil {
			log.Fatal(err)
		}
	case cleanupCmd.FullCommand():
		if err := cleanup(directory, force); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	Run(kingpin.New("git-branches", "Manage branches with interfaces"))
}
