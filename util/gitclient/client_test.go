package gitclient

import (
	"os"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/tmax-cloud/cd-operator/internal/utils"
)

func TestClone(t *testing.T) {
	url := "https://github.com/tmax-cloud/cd-example-apps"
	path := "/tmp/test-" + utils.RandomString(5)

	defer os.RemoveAll(path)

	err := Clone(url, path)
	assert.Equal(t, err, nil)
}
