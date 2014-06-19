// +build !windows

package shellwords

import (
	"errors"
	"os"
	"os/exec"
)

func shellRun(line string) (string, error) {
	shell := os.Getenv("SHELL")
	b, err := exec.Command(shell, "-c", line).Output()
	if err != nil {
		return "", errors.New(err.Error() + ":" + string(b))
	}
	return string(b), nil
}
