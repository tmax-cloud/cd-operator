package gitclient

import (
	"os"

	"github.com/go-git/go-git/v5"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("git-client")

// Clone the given repository to the given directory
func Clone(url string, path string) error {
	log.Info("git clone " + url)

	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})

	if err != nil {
		log.Error(err, "git.PlainClone failed..")
		return err
	}

	return nil
}
