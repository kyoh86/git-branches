package main

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/kyoh86/git-branches/gitbranches"
)

var (
	version = "snapshot"
	commit  = "snapshot"
	date    = "snapshot"
)

func main() {
  app := kingpin.New("git-branches", "Manage branches with interfaces")
	app.Version(fmt.Sprintf("%s-%s (%s)", version, commit, date))
	gitbranches.Run(app)
}
