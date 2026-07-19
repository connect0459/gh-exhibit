// Command gh-exhibit exports a GitHub issue or pull request's full evidence
// trail (raw JSON, rendered Markdown, downloaded attachments) to local
// files, for use as a gh CLI extension.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"

	"github.com/connect0459/gh-exhibit/internal/presentation/cli"
	"github.com/connect0459/gh-exhibit/internal/registry"
)

func main() {
	os.Exit(run())
}

func run() int {
	args, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 2
	}

	repo, err := cli.ResolveRepo(args.Repo, repository.Current)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 2
	}

	exporter, err := registry.NewExportService(registry.Config{Host: repo.Host, OutputDir: args.OutputDir})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 2
	}

	return cli.RunExports(context.Background(), exporter, repo.Owner, repo.Name, args.OutputDir, args.Numbers, os.Stdout, os.Stderr)
}
