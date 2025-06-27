package main

import (
	"github.com/giantswarm/mcp-giantswarm-apps/cmd"
)

const version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
