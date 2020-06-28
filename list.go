package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/morikuni/aec"
)

func list(
	directory string,
	color bool,
	filters []string,
) error {
	branches, err := retrieveBranchList(directory)
	if err != nil {
		return fmt.Errorf("get branch list: %w", err)
	}

	filter := ParseFilter(filters)
	write, close := columnWriter(os.Stdout)
	defer close()

	formatRemote := formatRemoteFunc(color)
	formatAuthor := formatAuthorFunc(color)

	for _, branch := range branches {
		if !filter(branch) {
			continue
		}

		write([]string{
			formatRemote(branch),
			branch.Name,
			formatAuthor(branch),
			formatUpstream(branch),
		})
	}
	return nil
}

func formatUpstream(b *Branch) string {
	if b.Upstream == "" {
		return ""
	}
	if b.Living {
		return "=> " + b.Upstream
	}
	return "DEAD"
}

func formatRemoteFunc(color bool) func(*Branch) string {
	if color {
		return func(b *Branch) string {
			switch b.Remote {
			case "":
				return aec.BlueF.Apply("local")
			default:
				return aec.RedF.Apply(b.Remote)
			}
		}
	}
	return func(b *Branch) string {
		if b.Remote == "" {
			return "local"
		}
		return b.Remote
	}
}

func formatAuthorFunc(color bool) func(*Branch) string {
	//HACK: 自分かどうかで色分け
	if color {
		return func(b *Branch) string {
			return aec.GreenF.Apply(b.Committer)
		}
	}
	return func(b *Branch) string {
		return b.Committer
	}
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
