package gitclient

import (
	"os"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/tmax-cloud/cd-operator/internal/utils"
)

func TestClone(t *testing.T) {
	url := "https://github.com/tmax-cloud/cd-example-apps"
	localPath := "/tmp/test-" + utils.RandomString(5)
	revision := "main"

	defer os.RemoveAll(localPath)

	err := Clone(url, localPath, revision)
	assert.Equal(t, err, nil)
}
