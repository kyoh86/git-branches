package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/Sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"
	"github.com/kyoh86/git-branches/log"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
)

type arguments struct {
	//TODO: support JSON format, pretty template style

	ExcludeCurrent bool    `short:"X" long:"exclude-current" description:"Exclude current branch"`
	WorkingDir     *string `short:"W" long:"working-dir" description:"Run as if git was started in <path> instead of the current working directory."`
	Color          bool    `long:"color" description:"Output with ANSI colors."`
}

const (
	remotesPrefix = "remotes/"
	currentCursor = "* "
	headCommit    = "->"
)

func main() {
	log.InitLogger()

	var args arguments
	_, err := flags.Parse(&args)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to parse argument")
	}

	branches, commits, err := retrieveBranchList(args.WorkingDir)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get branch list")
	}

	authors, err := retrieveCommitAuthors(commits, args.WorkingDir)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get commit-author list")
	}

	write, close := columnWriter(os.Stdout)
	defer close()

	colorRemote := colorRemoteFunc(args.Color)
	colorAuthor := colorAuthorFunc(args.Color)
	for _, branch := range branches {
		if args.ExcludeCurrent && branch.Current {
			continue
		}
		//HACK: 項目の並び替え、有無を選択できるようにする
		upstream := branch.Upstream
		if upstream != "" {
			upstream = "=> " + upstream
		}
		write([]string{
			colorRemote(branch.Remote),
			branch.Name,
			colorAuthor(authors[branch.Commit]),
			upstream,
		})
	}
}

func colorRemoteFunc(color bool) func(string) string {
	if color {
		return func(s string) string {
			if s == "" {
				return aurora.Blue("local").String()
			}
			return aurora.Red(s).String()
		}
	}
	return func(s string) string {
		if s == "" {
			return "local"
		}
		return s
	}
}

func colorAuthorFunc(color bool) func(string) string {
	//HACK: 自分かどうかで色分け
	if color {
		return func(s string) string {
			return aurora.Green(s).String()
		}
	}
	return func(s string) string {
		return s
	}
}

// Branch contains information of a branch
type Branch struct {
	Current  bool
	Remote   string
	Name     string
	Commit   string
	Upstream string
}

// FullName gets path of the origin and name
func (b Branch) FullName() string {
	if b.Remote == "" {
		return b.Name
	}
	return b.Remote + "/" + b.Name
}

func retrieveBranchList(currentDir *string) ([]Branch, []string, error) {
	output, err := callGit([]string{"branch", "--list", "--all", "-vv"}, currentDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to call git-branch")
	}

	commitMap := map[string]struct{}{}
	commits := []string{}
	branches := []Branch{}
	followees := map[string]struct{}{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for line := 0; scanner.Scan(); line++ {
		text := scanner.Text()
		branch, err := parseBranchText(line, text)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("failed to parse branch text at line %d (%s)", line, text))
		}
		if branch == nil {
			continue
		}

		if branch.Upstream != "" {
			followees[branch.Upstream] = struct{}{}
		}
		if _, ok := followees[branch.FullName()]; ok {
			logrus.WithField("branch", *branch).Debug("ignore followed branch")
			continue
		}
		if _, ok := commitMap[branch.Commit]; !ok {
			commitMap[branch.Commit] = struct{}{}
			commits = append(commits, branch.Commit)
		}
		branches = append(branches, *branch)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse git-branch output")
	}
	return branches, commits, nil
}

func retrieveCommitAuthors(commits []string, currentDir *string) (map[string]string, error) {
	//HACK: 表示する情報をan(authors' name)ではなくae(authors' email)も選べるようにする
	output, err := callGit(append([]string{"show", "--format=%h:%an", "--no-patch"}, commits...), currentDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call git-show")
	}

	authors := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for line := 0; scanner.Scan(); line++ {
		text := scanner.Text()
		terms := strings.SplitN(text, ":", 2)
		logrus.WithField("commit", terms[0]).WithField("author", terms[1]).Debug("found commit info")
		authors[terms[0]] = terms[1]
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to parse git-show output")
	}
	return authors, nil
}

func callGit(args []string, currentDir *string) ([]byte, error) {
	if currentDir != nil {
		args = append([]string{"-C", *currentDir}, args...)
	}
	logrus.WithField("args", strings.Join(args, " ")).Debug("git-args")
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, err
	}
	logrus.WithField("output", string(output)).Debug("git responded")
	return output, err
}

func parseBranchText(line int, text string) (*Branch, error) {
	var current bool
	cursor, row := text[:2], text[2:]
	fields := strings.Fields(row)
	if len(fields) < 3 {
		return nil, errors.New("shortage of fields (<3)")
	}
	name, commit, upstream := fields[0], fields[1], fields[2]

	if commit == headCommit {
		logrus.WithField("headCommit", headCommit).Debug("ignore HEAD link branch")
		return nil, nil
	}

	if strings.HasPrefix(upstream, "[") &&
		(strings.HasSuffix(upstream, ":") ||
			strings.HasSuffix(upstream, "]")) {
		upstream = upstream[1 : len(upstream)-1]
	} else {
		upstream = ""
	}

	if cursor == currentCursor {
		current = true
	}

	var remote string
	if strings.HasPrefix(name, remotesPrefix) {
		name = name[len(remotesPrefix):]
		paths := strings.SplitN(name, "/", 2)

		remote, name = paths[0], paths[1]
		upstream = ""
	}
	return &Branch{
		Current:  current,
		Remote:   remote,
		Name:     name,
		Commit:   commit,
		Upstream: upstream,
	}, nil
}

func columnWriter(io.Writer) (write func([]string), close func()) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	c := csv.NewWriter(w)
	c.Comma = '\t'
	c.UseCRLF = false

	return func(field []string) {
			escaped := make([]string, 0, len(field))
			for _, f := range field {
				escaped = append(escaped, strings.Replace(f, " ", "_", -1))
			}
			c.Write(escaped)
		}, func() {
			c.Flush()
			w.Flush()
		}
}
