package main

import "fmt"

func cleanup(force bool, directory string) error {
	branches, err := retrieveBranchList(directory)
	if err != nil {
		return fmt.Errorf("get branch list: %w", err)
	}
	for _, branch := range branches {
		if branch.Living {
			continue
		}
		_, err := callGit([]string{"-d", branch.Name}, directory)
		if err != nil {
			return err
		}
		//TODO: ask and -D
	}

	return nil
}
