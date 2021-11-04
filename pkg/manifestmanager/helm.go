package manifestmanager

import (
	"os"

	gohelm "github.com/mittwald/go-helm-client"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/util/gitclient"
	"github.com/tmax-cloud/cd-operator/util/helmclient"
)

type helmManager struct {
	helmClient *helmclient.Client
}

func NewHelmManager() ManifestManager {
	opt := &gohelm.Options{
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          true,
	}

	goHelmClient, err := gohelm.New(opt)
	if err != nil {
		panic(err)
	}
	return &helmManager{
		helmClient: &helmclient.Client{Client: goHelmClient},
	}
}

func (m *helmManager) Sync(app *cdv1.Application, forced bool) error {
	if err := m.gitRepoClone(app); err != nil {
		return err
	}

	if forced || app.Spec.SyncPolicy.AutoSync {
		if err := m.installHelmChart(app); err != nil {
			return err
		}
	}
	return nil
}

func (m *helmManager) Clear(app *cdv1.Application) error {
	return nil
}

func (m *helmManager) gitRepoClone(app *cdv1.Application) error {
	repo := app.Spec.Source.RepoURL
	revision := app.Spec.Source.TargetRevision

	localPath := "/tmp/repo-" + utils.RandomString(5)

	app.Spec.Source.Helm.ClonedRepoPath = localPath

	err := gitclient.Clone(repo, localPath, revision)
	if err != nil {
		panic(err)
	}

	return nil
}

func (m *helmManager) installHelmChart(app *cdv1.Application) error {
	// 로컬에 저장된 경로를 이용하여 chart install
	randomString := utils.RandomString(5)
	releaseName := "release-" + randomString
	app.Spec.Source.Helm.ReleaseName = releaseName

	chartPath := app.Spec.Source.Path
	chartLocalPath := app.Spec.Source.Helm.ClonedRepoPath + "/" + chartPath

	var namespace string
	if app.Spec.Destination.Namespace == "" {
		namespace = "default"
	} else {
		namespace = app.Spec.Destination.Namespace
	}

	err := m.helmClient.InstallChart(releaseName, chartLocalPath, namespace)
	if err != nil {
		panic(err)
	}

	// 로컬에 clone된 Repo 디렉토리 삭제
	os.RemoveAll(app.Spec.Source.Helm.ClonedRepoPath)
	return nil
}

func (m *helmManager) uninstallRelease(app *cdv1.Application) error {
	err := m.helmClient.UninstallReleaseByName(app.Spec.Source.Helm.ReleaseName)
	if err != nil {
		panic(err)
	}

	return nil
}
