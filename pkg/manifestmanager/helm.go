package manifestmanager

import (
	"context"
	"os"

	gohelm "github.com/mittwald/go-helm-client"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/cluster"
	"github.com/tmax-cloud/cd-operator/util/gitclient"
	"github.com/tmax-cloud/cd-operator/util/helmclient"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type helmManager struct {
	DefaultCli client.Client
	context.Context
	helmClient *helmclient.Client
}

func NewHelmManager(ctx context.Context, cli client.Client) ManifestManager {
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
		Context:    ctx,
		DefaultCli: cli,
		helmClient: &helmclient.Client{
			Client: goHelmClient,
		},
	}
}

func (m *helmManager) Sync(app *cdv1.Application, forced bool) error {
	if err := m.gitRepoClone(app); err != nil {
		return err
	}

	chartSpec := setChartSpec(app)

	if err := m.setTargetClient(app); err != nil {
		log.Error(err, "setTargetClient failed..")
		return err
	}

	oldDeployResources, err := getDeployResourceList(m.DefaultCli, app)
	if err != nil {
		log.Error(err, "GetDeployResourceList failed")
		return err
	}

	updatedDeployResources := make(map[string]*cdv1.DeployResource)

	manifestRawobjs, err := m.objectFromManifest(chartSpec, app)
	if err != nil {
		log.Error(err, "Get object from manifest failed..")
		return err
	}

	for _, manifestRawobj := range manifestRawobjs {
		updatedDeployResource, err := updateDeployResource(m.DefaultCli, manifestRawobj, app)
		if err != nil {
			log.Error(err, "NewDeployResource failed..")
			return err
		}
		updatedDeployResources[updatedDeployResource.Name] = updatedDeployResource
	}

	for _, oldDeployResource := range oldDeployResources.Items {
		if updatedDeployResources[oldDeployResource.Name] == nil {
			if err := deleteDeployResource(m.DefaultCli, &oldDeployResource); err != nil {
				log.Error(err, "DeleteDeployResource failed..")
				return err
			}
		}
	}

	if forced || app.Spec.SyncPolicy.AutoSync {
		if _, err := m.installHelmChart(chartSpec, app, false); err != nil {
			return err
		}
		app.Status.Sync.Status = cdv1.SyncStatusCodeSynced
	}
	return nil
}

func (m *helmManager) Clear(app *cdv1.Application) error {
	if err := m.setTargetClient(app); err != nil {
		return err
	}

	if err := m.uninstallRelease(app); err != nil {
		return err
	}

	deployedResourceList, err := getDeployResourceList(m.DefaultCli, app)
	if err != nil {
		return err
	}

	for _, deployedResource := range deployedResourceList.Items {
		if err := m.clearDeployResource(&deployedResource); err != nil {
			return err
		}
	}

	return nil
}

func (m *helmManager) gitRepoClone(app *cdv1.Application) error {
	repo := app.Spec.Source.RepoURL
	revision := app.Spec.Source.TargetRevision
	localPath := "/tmp/repo-" + app.Name + "-" + app.Namespace

	if err := gitclient.Clone(repo, localPath, revision); err != nil {
		return err
	}

	return nil
}

func (m *helmManager) objectFromManifest(chartSpec *gohelm.ChartSpec, app *cdv1.Application) ([]*unstructured.Unstructured, error) {
	var manifestRawObjs []*unstructured.Unstructured

	manifest, err := m.installHelmChart(chartSpec, app, true)
	if err != nil {
		return nil, err
	}

	stringYAMLManifests := splitMultipleObjectsYAML([]byte(manifest))

	for _, stringYAMLManifest := range stringYAMLManifests {
		byteYAMLManifest := []byte(stringYAMLManifest)

		bytes, err := yaml.YAMLToJSON(byteYAMLManifest)
		if err != nil {
			log.Error(err, "YAMLToJSON failed..")
			return nil, err
		}

		if string(bytes) == "null" {
			continue
		}

		if err := m.setTargetClient(app); err != nil {
			log.Error(err, "setTargetClient failed..")
			return nil, err
		}

		rawExt := &runtime.RawExtension{Raw: bytes}
		manifestRawObj, err := bytesToUnstructuredObject(rawExt)
		if err != nil {
			log.Error(err, "BytesToUnstructuredObject failed..")
			return nil, err
		}

		if len(manifestRawObj.GetNamespace()) == 0 {
			manifestRawObj.SetNamespace(app.Spec.Destination.Namespace)
		}
		manifestRawObjs = append(manifestRawObjs, manifestRawObj)
	}
	return manifestRawObjs, nil
}

func (m *helmManager) installHelmChart(chartSpec *gohelm.ChartSpec, app *cdv1.Application, dryRun bool) (string, error) {
	chartSpec.DryRun = dryRun
	manifest, err := m.helmClient.InstallChart(chartSpec)
	if err != nil {
		panic(err)
	}

	// 로컬에 clone된 Repo 디렉토리 삭제
	if !dryRun {
		os.RemoveAll("/tmp/repo-" + app.Name + "-" + app.Namespace)
	}
	return manifest, nil
}

func (m *helmManager) uninstallRelease(app *cdv1.Application) error {
	log.Info(app.Spec.Source.Helm.ReleaseName)
	err := m.helmClient.UninstallReleaseByName(app.Name + "-" + app.Namespace)
	if err != nil {
		return err
	}

	return nil
}

func setChartSpec(app *cdv1.Application) *gohelm.ChartSpec {
	// 로컬에 저장된 경로를 이용하여 chart install
	releaseName := app.Name + "-" + app.Namespace
	chartPath := app.Spec.Source.Path
	chartLocalPath := "/tmp/repo-" + app.Name + "-" + app.Namespace + "/" + chartPath

	var namespace string
	if app.Spec.Destination.Namespace == "" {
		namespace = "default"
	} else {
		namespace = app.Spec.Destination.Namespace
	}

	return &gohelm.ChartSpec{
		ReleaseName: releaseName,
		ChartName:   chartLocalPath,
		Namespace:   namespace,
		UpgradeCRDs: true,
		Wait:        false,
	}
}

func (m *helmManager) setTargetClient(app *cdv1.Application) error {
	if app.Spec.Destination.Name != "" {
		cfg, err := cluster.GetApplicationClusterConfig(m.Context, m.DefaultCli, app)
		if err != nil {
			log.Error(err, "GetConfig failed..")
			return err
		}

		opt := &gohelm.RestConfClientOptions{
			Options: &gohelm.Options{
				RepositoryCache:  "/tmp/.helmcache",
				RepositoryConfig: "/tmp/.helmrepo",
				Debug:            true,
				Linting:          true,
			},
			RestConfig: cfg,
		}
		cli, err := gohelm.NewClientFromRestConf(opt)
		if err != nil {
			return err
		}
		m.helmClient.Client = cli
	} else {
		opt := &gohelm.Options{
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            true,
			Linting:          true,
		}
		cli, err := gohelm.New(opt)
		if err != nil {
			return err
		}
		m.helmClient.Client = cli
	}
	return nil
}

func (m *helmManager) clearDeployResource(deployResource *cdv1.DeployResource) error {
	if err := m.DefaultCli.Delete(m.Context, deployResource); err != nil {
		log.Error(err, "Delete DeployResource error..")
		return err
	}
	return nil
}
