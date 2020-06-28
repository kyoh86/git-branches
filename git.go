package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func callGitWithStdout(args []string, workingDir string) ([]byte, error) {
	var stdout bytes.Buffer
	err := callGit(args, workingDir, func(cmd *exec.Cmd) {
		cmd.Stdout = &stdout
	})
	return stdout.Bytes(), err
}

func callGitWithStderr(args []string, workingDir string) ([]byte, error) {
	var stderr bytes.Buffer
	err := callGit(args, workingDir, func(cmd *exec.Cmd) {
		cmd.Stderr = &stderr
	})
	return stderr.Bytes(), err
}

func callGit(args []string, workingDir string, cmdWrap ...func(*exec.Cmd)) error {
	if workingDir != "" {
		args = append([]string{"-C", workingDir}, args...)
	}
	cmd := exec.Command("git", args...)
	for _, w := range cmdWrap {
		w(cmd)
	}
	return cmd.Run()
}

func retrieveBranchListParams() []string {
	return []string{"for-each-ref", "--format",
		strings.Join([]string{
			/**/ "%(if:notequals=refs/stash)%(refname:rstrip=-2)%(then)",
			/*  */ "%(if:notequals=refs/tags)%(refname:rstrip=-2)%(then)",
			/*    */ "%(if:notequals=HEAD)%(refname:lstrip=3)%(then)",
			/*      */ "%(HEAD)%09%(refname:lstrip=2)%09%(authorname)%09",
			/*      */ "%(if)%(upstream)%(then)",
			/*        */ "%(upstream:lstrip=2)",
			/*      */ "%(end)",
			/*      */ "%09%(committerdate:format-local:%Y/%m/%d %H:%M:%S)",
			/*    */ "%(end)",
			/*  */ "%(end)",
			/**/ "%(end)",
		}, "")}
}

func retrieveBranchList(workingDir string) ([]*Branch, error) {
	params := retrieveBranchListParams()
	stdout , err := callGitWithStdout(params, workingDir)
	if err != nil {
		return nil, fmt.Errorf("call git-branch: %w", err)
	}

	branches := []*Branch{}
	followees := map[string]*Branch{}
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for line := 0; scanner.Scan(); line++ {
		text := scanner.Text()
		// HEAD?, name, committer, upstream, time
		fields := strings.Split(text, "\t")
		if len(fields) < 5 {
			continue
		}

		branch := &Branch{
			Current:   fields[0] == "*",
			Committer: fields[2],
			Upstream:  fields[3],
		}

		names := strings.SplitN(fields[1], "/", 2)
		if len(names) == 2 {
			branch.Remote = names[0]
			branch.Name = names[1]
		} else {
			branch.Name = names[0]
		}
		if branch.Upstream != "" {
			followees[branch.Upstream] = branch
		}
		if follower, ok := followees[branch.FullName()]; ok {
			follower.Living = true
			continue
		}
		branches = append(branches, branch)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse git-branch output: %w", err)
	}
	return branches, nil
}
