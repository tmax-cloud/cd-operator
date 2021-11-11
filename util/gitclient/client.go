package gitclient

import (
	"os"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("git-client")

// Clone the given repository to the given directory
func Clone(url string, localPath string, revision string) (*gogit.Repository, error) {
	log.Info("git clone " + url)

	// TODO : tag(NewTagReferenceName)나 commit SHA 일 경우도 지원해줘야 함
	referenceName := plumbing.NewBranchReferenceName(revision)
	// comment : 이미 존재하는 폴더에 클론을 받으면 ErrRepositoryAlreadyExists 오류 출력
	repo, err := gogit.PlainClone(localPath, false, &gogit.CloneOptions{
		URL:           url,
		ReferenceName: referenceName,
		Progress:      os.Stdout,
	})

	if err != nil {
		log.Error(err, "git.PlainClone failed..")
		return nil, err
	}

	return repo, nil
}

// Open opens a git repository from the given path
func Open(localRepoPath string) (*gogit.Repository, error) {
	repo, err := gogit.PlainOpen(localRepoPath)
	if err != nil {
		log.Error(err, "gogit.PlainOpen failed..")
		return nil, err
	}

	return repo, err
}

// Fetch fetches refenences along with the objects necessary to complete their histories,
// from the remote named as FetchOptions.RemoteName
func Fetch(repo *gogit.Repository) error {
	err := repo.Fetch(&gogit.FetchOptions{
		RemoteName: "origin", // TODO : 다른 경우도 고민해볼 것
		Progress:   os.Stdout,
	})
	if err != nil {
		// 이미 최신 상태라면 already up-to-date 오류 리턴
		if err != gogit.NoErrAlreadyUpToDate {
			log.Error(err, "repo.Fetch failed..")
			return err
		}
	}
	return nil
}

// Pull incorporates changes from a remote repository into the current branch
func Pull(repo *gogit.Repository) error {
	worktree, err := repo.Worktree()
	if err != nil {
		panic(err)
	}

	err = worktree.Pull(&gogit.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	})

	// 이미 최신 상태라면 already up-to-date라는 오류 리턴
	if err != nil {
		if err != gogit.NoErrAlreadyUpToDate {
			log.Error(err, "worktree.Pull failed..")
			return err
		}
	}

	return nil
}
