package gitclient

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("git-client")

// Clone the given repository to the given directory
func Clone(url string, localPath string, revision string) error {
	log.Info("git clone " + url)

	// TODO : tag(NewTagReferenceName)나 commit SHA 일 경우도 지원해줘야 함
	referenceName := plumbing.NewBranchReferenceName(revision)
	_, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: referenceName,
		Progress:      os.Stdout,
	})

	if err != nil {
		log.Error(err, "git.PlainClone failed..")
		return err
	}

	return nil
}
