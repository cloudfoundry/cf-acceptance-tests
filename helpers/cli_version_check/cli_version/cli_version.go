package cli_version

import (
	"fmt"
	"os"
	"os/exec"
)

type CliVersion interface {
	GetVersion() string
}

type cliVersion struct{}

func NewCliVersion() CliVersion {
	return &cliVersion{}
}

func (c *cliVersion) GetVersion() string {
	output, err := exec.Command("cf", "-v").Output()
	if err != nil {
		fmt.Println("Error trying to determine CF CLI version:", err)
		os.Exit(1)
	}

	return string(output)
}
