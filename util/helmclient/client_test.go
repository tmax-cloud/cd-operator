package helmclient

import (
	"context"
	"os"
	"testing"

	gohelm "github.com/mittwald/go-helm-client"
	"github.com/stretchr/testify/require"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/util/gitclient"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestInstallChart(t *testing.T) {
	opt := &gohelm.Options{
		Namespace:        "default",
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          true,
	}

	helmClient, err := gohelm.New(opt)
	require.NoError(t, err)

	testHelmClient := &Client{Client: helmClient}

	url := "https://github.com/tmax-cloud/cd-example-apps"
	randomString := utils.RandomString(5)
	path := "/tmp/test-" + randomString
	revision := "main"

	defer os.RemoveAll(path)

	// 1. 로컬에 helm manifest의 git repo clone
	_, err = gitclient.Clone(url, path, revision)
	require.Equal(t, err, nil)

	// 2. 로컬에 저장된 경로를 이용하여 chart install
	releaseName := "test-" + randomString
	chartPath := path + "/helm-guestbook"
	namespace := "default"
	chartSpec := &gohelm.ChartSpec{
		ReleaseName: releaseName,
		ChartName:   chartPath,
		Namespace:   namespace,
		UpgradeCRDs: true,
		Wait:        false,
	}

	_, err = testHelmClient.InstallChart(chartSpec)
	require.Equal(t, err, nil)

	defer func() {
		err := testHelmClient.UninstallReleaseByName(releaseName)
		require.NoError(t, err)
	}()

	cfg, err := config.GetConfig()
	require.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)

	deploy, err := clientset.AppsV1().Deployments("default").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	require.Equal(t, err, nil)
	svc, err := clientset.CoreV1().Services("default").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	require.Equal(t, err, nil)

	require.NotEqual(t, deploy, nil)
	require.NotEqual(t, svc, nil)
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
	_, err := gitclient.Clone(url, path, revision)
	require.Equal(t, err, nil)

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
	require.Equal(t, err, nil)

	defer func() {
		err := testHelmClient.UninstallReleaseByCLI(releaseName, namespace)
		require.NoError(t, err)
	}()

	cfg, err := config.GetConfig()
	require.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)

	deploy, err := clientset.AppsV1().Deployments("testjh").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	require.Equal(t, err, nil)
	svc, err := clientset.CoreV1().Services("testjh").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	require.Equal(t, err, nil)

	require.NotEqual(t, deploy, nil)
	require.NotEqual(t, svc, nil)
}
