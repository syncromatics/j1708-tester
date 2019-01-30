package main

import (
	"github.com/syncromatics/j1708-tester/cmd/j1708-tester/cmd"

	_ "github.com/syncromatics/j1708-tester/internal/web/statik"
)

func main() {
	cmd.Execute()
}
