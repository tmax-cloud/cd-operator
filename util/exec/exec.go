package exec

import (
	"fmt"
	"os/exec"
)

// TODO - Timeout 관련된 로직 추가할 것

func Run(cmd *exec.Cmd) (string, error) {
	out, err := cmd.CombinedOutput() // Run and return output & error

	if err != nil {
		fmt.Printf((string(out)))
		return string(out), err
	}

	return string(out), nil
}
