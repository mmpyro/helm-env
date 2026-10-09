package commands

import (
	"fmt"

	"github.com/user/helm-env/internal/config"
)

// WhichHelp prints help for the which command.
func WhichHelp() {
	fmt.Println(`Usage: helm-env which

Print the full path to the active helm binary that would be used based on
version resolution.`)
}

// Which prints the absolute path to the active helm binary.
func Which() error {
	if err := config.RequireInit(); err != nil {
		return err
	}
	version, err := config.ResolveConcreteVersion()
	if err != nil {
		return err
	}
	binaryPath, err := config.GetBinaryPath(version)
	if err != nil {
		return err
	}
	fmt.Println(binaryPath)
	return nil
}
