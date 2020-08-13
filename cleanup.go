package main

import (
	"fmt"
	"strings"

	"github.com/kyoh86/ask"
	"github.com/morikuni/aec"
)

func cleanup(directory string, force bool) error {
	branches, err := retrieveBranchList(directory)
	if err != nil {
		return fmt.Errorf("get branch list: %w", err)
	}
	for _, branch := range branches {
		if branch.Upstream == "" {
			continue
		}
		if branch.Living {
			continue
		}

		stderr, err := callGitWithStderr([]string{"branch", "-d", branch.Name}, directory)
		if err != nil {
			errMsg := string(stderr)
			if !strings.Contains(errMsg, "If you are sure you want to delete it, run 'git branch -D") {
				return fmt.Errorf("calling git -d: %s", errMsg)
			}
		}

		var yes bool
		if force {
			yes = true
		} else if err := ask.Message(fmt.Sprintf(
			"%s The branch '%s' is not fully merged. Are you sure you want to delete it? [Y/n]",
			aec.BlueF.With(aec.Bold).Apply("?"),
			aec.YellowF.With(aec.Bold).Apply(branch.Name),
		)).YesNoVar(&yes).Do(); err != nil {
			return fmt.Errorf("ask: %w", err)
		}

		if yes {
			stderr, err := callGitWithStderr([]string{"branch", "-D", branch.Name}, directory)
			if err != nil {
				return fmt.Errorf("calling git -D: %s", string(stderr))
			}
		}
	}

	return nil
}
