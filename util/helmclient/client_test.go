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

	testHelmClient := &Client{client: helmClient}

	url := "https://github.com/tmax-cloud/cd-example-apps"
	randomString := utils.RandomString(5)
	path := "/tmp/test-" + randomString

	defer os.RemoveAll(path)

	// 1. 로컬에 helm manifest의 git repo clone
	err = gitclient.Clone(url, path)
	assert.Equal(t, err, nil)

	// 2. 로컬에 저장된 경로를 이용하여 chart install
	releaseName := "test-" + randomString
	chartPath := path + "/helm-guestbook"
	namespace := "default"
	err = testHelmClient.InstallChart(releaseName, chartPath, namespace)
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

	deploy, _ := clientset.AppsV1().Deployments("default").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})
	svc, _ := clientset.CoreV1().Services("default").Get(context.Background(), releaseName+"-helm-guestbook", v1.GetOptions{})

	assert.NotEqual(t, deploy, nil)
	assert.NotEqual(t, svc, nil)
}
