package manifestmanager

import (
	"os"
	"testing"

	gohelm "github.com/mittwald/go-helm-client"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/util/helmclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type gitRepoCloneTestCase struct {
	app *cdv1.Application
}

func TestGitRepoClone(t *testing.T) {
	tc := map[string]gitRepoCloneTestCase{
		"helm-app-1": {
			app: &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-app-1",
					Namespace: "default",
				},
				Spec: cdv1.ApplicationSpec{
					Source: cdv1.ApplicationSource{
						RepoURL:        "https://github.com/tmax-cloud/cd-example-apps",
						Path:           "helm-guestbook",
						TargetRevision: "main",
						Type:           "Helm",
						Helm:           &cdv1.ApplicationSourceHelm{}, // TODO : Default로 nil이 됨. 추후 소스 타입 체킹할 떄, 할당해줄 것
					},
				},
			},
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	opt := &gohelm.Options{
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          true,
	}

	goHelmClient, err := gohelm.New(opt)
	if err != nil {
		require.NoError(t, err)
	}
	m := &helmManager{helmClient: &helmclient.Client{Client: goHelmClient}}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			err := m.gitRepoClone(c.app)
			os.RemoveAll("/tmp/repo-" + c.app.Name + "-" + c.app.Namespace)
			require.NoError(t, err)
		})
	}
}

type InstallHelmChartTestCase struct {
	app *cdv1.Application
}

func TestInstallHelmChart(t *testing.T) {
	tc := map[string]InstallHelmChartTestCase{
		"helm-app-1": {
			app: &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-app-1",
					Namespace: "default",
				},
				Spec: cdv1.ApplicationSpec{
					Source: cdv1.ApplicationSource{
						RepoURL:        "https://github.com/tmax-cloud/cd-example-apps",
						Path:           "helm-guestbook",
						TargetRevision: "main",
						Helm:           &cdv1.ApplicationSourceHelm{},
					},
				},
			},
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	opt := &gohelm.Options{
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          true,
	}

	goHelmClient, err := gohelm.New(opt)
	if err != nil {
		require.NoError(t, err)
	}
	m := &helmManager{helmClient: &helmclient.Client{Client: goHelmClient}}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			err := m.gitRepoClone(c.app)
			require.NoError(t, err)

			chartSpec := setChartSpec(c.app)
			defer func() {
				err := m.uninstallRelease(c.app)
				require.NoError(t, err)
			}()
			_, err = m.installHelmChart(chartSpec, c.app, false)
			require.NoError(t, err)
		})
	}
}
