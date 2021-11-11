package gitclient

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tmax-cloud/cd-operator/internal/utils"
)

const (
	testRepoURL = "https://github.com/tmax-cloud/cd-example-apps"
)

func TestClone(t *testing.T) {
	localPath := "/tmp/test-" + utils.RandomString(5)
	revision := "main"
	t.Log(localPath)

	defer os.RemoveAll(localPath)

	repo, err := Clone(testRepoURL, localPath, revision)
	require.NotEmpty(t, repo)
	require.NoError(t, err)
}

type openTestCase struct {
	localRepoPath string

	expectedErrOccur bool
	expectedErrMsg   string
}

func TestOpen(t *testing.T) {
	localPath1 := "/tmp/test-" + utils.RandomString(5)
	localPath2 := "/tmp/test-" + utils.RandomString(5)
	revision := "main"

	// Clone for ValidRepo Case
	_, err := Clone(testRepoURL, localPath1, revision)
	require.NoError(t, err)
	defer os.RemoveAll(localPath1)

	// Just empty dir for InvalidRepo Case
	err = os.MkdirAll(localPath2, os.ModePerm)
	require.NoError(t, err)
	defer os.RemoveAll(localPath2)

	tc := map[string]openTestCase{
		"ValidRepo": {
			localRepoPath: localPath1,

			expectedErrOccur: false,
		},
		"InvalidRepo": {
			localRepoPath: localPath2,

			expectedErrOccur: true,
			expectedErrMsg:   "repository does not exist",
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			repo, err := Open(c.localRepoPath)
			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NotEmpty(t, repo)
				require.NoError(t, err)
			}
		})
	}
}

func TestFetch(t *testing.T) {
	localPath := "/tmp/test-" + utils.RandomString(5)
	revision := "main"
	t.Log(localPath)

	defer os.RemoveAll(localPath)

	repo, err := Clone(testRepoURL, localPath, revision)
	require.NotEmpty(t, repo)
	require.NoError(t, err)

	err = Fetch(repo)
	require.NoError(t, err)
}

func TestPull(t *testing.T) {
	localPath := "/tmp/test-" + utils.RandomString(5)
	revision := "main"
	t.Log(localPath)

	defer os.RemoveAll(localPath)

	repo, err := Clone(testRepoURL, localPath, revision)
	require.NotEmpty(t, repo)
	require.NoError(t, err)

	err = Pull(repo)
	require.NoError(t, err)
}
