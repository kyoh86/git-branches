package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kingpin"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetOutput(os.Stderr)
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	var args struct {
		ExcludeCurrent bool
		Directory      string
		Color          bool
		DeadOnly       bool
	}

	app := kingpin.New("git-branches", "Show each branch, upstream, author in git repository.")
	app.Flag("exclude-current", "Exclude current branch").Short('X').BoolVar(&args.ExcludeCurrent)
	app.Flag("directory", "Run as if git was started in <path> instead of the current working directory.").Short('C').StringVar(&args.Directory)
	app.Flag("color", "Output with ANSI colors.").BoolVar(&args.Color)
	app.Flag("dead-only", "Show dead branches only.").Short('D').BoolVar(&args.Color)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	branches, err := retrieveBranchList(args.Directory)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get branch list")
	}

	write, close := columnWriter(os.Stdout)
	defer close()

	colorRemote := colorRemoteFunc(args.Color)
	colorAuthor := colorAuthorFunc(args.Color)
	for _, branch := range branches {
		if args.ExcludeCurrent && branch.Current {
			continue
		}
		upstream := branch.Upstream
		if upstream != "" {
			upstream = "=> " + upstream
			if branch.UpstreamIsLiving {
				if args.DeadOnly {
					continue
				}
			} else {
				branch.Remote = "DEAD"
			}
		}
		write([]string{
			colorRemote(branch.Remote),
			branch.Name,
			colorAuthor(branch.Committer),
			upstream,
		})
	}
}

func colorRemoteFunc(color bool) func(string) string {
	if color {
		return func(s string) string {
			switch s {
			case "":
				return aurora.Blue("local").String()
			case "DEAD":
				return aurora.BrightYellow(s).String()
			default:
				return aurora.Red(s).String()
			}
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
	Current          bool
	Remote           string
	Name             string
	Upstream         string
	UpstreamIsLiving bool
	Committer        string
}

// FullName gets path of the origin and name
func (b Branch) FullName() string {
	if b.Remote == "" {
		return b.Name
	}
	return b.Remote + "/" + b.Name
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

func retrieveBranchList(currentDir string) ([]*Branch, error) {
	params := retrieveBranchListParams()
	output, err := callGit(params, currentDir)
	if err != nil {
		return nil, errors.Wrap(err, "call git-branch")
	}

	branches := []*Branch{}
	followees := map[string]*Branch{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
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
			logrus.WithField("branch", *branch).Debug("ignore followed branch")
			follower.UpstreamIsLiving = true
			continue
		}
		branches = append(branches, branch)
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "parse git-branch output")
	}
	return branches, nil
}

func callGit(args []string, currentDir string) ([]byte, error) {
	if currentDir != "" {
		args = append([]string{"-C", currentDir}, args...)
	}
	logrus.WithField("args", strings.Join(args, " ")).Debug("git-args")
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, err
	}
	logrus.WithField("output", string(output)).Debug("git responded")
	return output, err
}

func columnWriter(io.Writer) (write func([]string), close func()) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	c := csv.NewWriter(w)
	c.Comma = '\t'
	c.UseCRLF = false

	return func(field []string) {
			escaped := make([]string, 0, len(field))
			for _, f := range field {
				escaped = append(escaped, strings.Replace(f, "\t", "_", -1))
			}
			if err := c.Write(escaped); err != nil {
				panic(err)
			}
		}, func() {
			c.Flush()
			w.Flush()
		}
}
