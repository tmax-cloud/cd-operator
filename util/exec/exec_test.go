package exec

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testStdoutString = "test"
)

func TestRun(t *testing.T) {
	stdout, err := Run(exec.Command("echo", "test"))
	trimmedStdout := strings.TrimRight(stdout, "\n")
	require.Equal(t, trimmedStdout, testStdoutString)
	require.NoError(t, err)
}
