// Command fortyhours fills out a Productive.io timesheet and time off.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cbarber/fortyhours/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
