package helmclient

import (
	"context"
	"os"
	"testing"

	"github.com/bmizerany/assert"
	gohelm "github.com/mittwald/go-helm-client"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/util/gitclient"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestInstallChart(t *testing.T) {
	opt := &gohelm.Options{
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          true,
	}

	helmClient, err := gohelm.New(opt)
	if err != nil {
		panic(err)
	}

	testHelmClient := &Client{Client: helmClient}

	url := "https://github.com/tmax-cloud/cd-example-apps"
	randomString := utils.RandomString(5)
	path := "/tmp/test-" + randomString
	revision := "main"

	defer os.RemoveAll(path)

	// 1. 로컬에 helm manifest의 git repo clone
	err = gitclient.Clone(url, path, revision)
	assert.Equal(t, err, nil)

	// 2. 로컬에 저장된 경로를 이용하여 chart install
	releaseName := "test-" + randomString
	chartPath := path + "/helm-guestbook"
	namespace := "test-1109"
	chartSpec := &gohelm.ChartSpec{
		ReleaseName: releaseName,
		ChartName:   chartPath,
		Namespace:   namespace,
		UpgradeCRDs: true,
		Wait:        false,
	}

	_, err = testHelmClient.InstallChart(chartSpec)
	assert.Equal(t, err, nil)

	defer func() {
		err := testHelmClient.UninstallReleaseByName(releaseName)
		if err != nil {
			panic(err)
		}
	}()

	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	deploy, err := clientset.AppsV1().Deployments("default").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	assert.Equal(t, err, nil)
	svc, err := clientset.CoreV1().Services("default").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	assert.Equal(t, err, nil)

	assert.NotEqual(t, deploy, nil)
	assert.NotEqual(t, svc, nil)
}

func TestInstallChartByCLI(t *testing.T) {
	// TODO : 네임스페이스가 없는 경우도 테스트 코드 짤 것 (INSTALLATION FAILED: create: failed to create: namespaces "testtesttest" not found)
	testHelmClient := &Client{}

	url := "https://github.com/tmax-cloud/cd-example-apps"
	randomString := utils.RandomString(5)
	path := "/tmp/test-" + randomString
	revision := "main"

	defer os.RemoveAll(path)

	// 1. 로컬에 helm manifest의 git repo clone
	err := gitclient.Clone(url, path, revision)
	assert.Equal(t, err, nil)

	// 2. 로컬에 저장된 경로를 이용하여 chart install
	releaseName := "test-" + randomString
	chartPath := path + "/helm-guestbook"
	namespace := "testjh"
	chartSpec := &gohelm.ChartSpec{
		ReleaseName: releaseName,
		ChartName:   chartPath,
		Namespace:   namespace,
		UpgradeCRDs: true,
		Wait:        false,
	}

	err = testHelmClient.InstallChartByCLI(chartSpec)
	assert.Equal(t, err, nil)

	defer func() {
		err := testHelmClient.UninstallReleaseByCLI(releaseName, namespace)
		if err != nil {
			panic(err)
		}
	}()

	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	deploy, err := clientset.AppsV1().Deployments("testjh").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	assert.Equal(t, err, nil)
	svc, err := clientset.CoreV1().Services("testjh").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	assert.Equal(t, err, nil)

	assert.NotEqual(t, deploy, nil)
	assert.NotEqual(t, svc, nil)
}
