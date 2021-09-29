package utils

import (
	"testing"

	"github.com/bmizerany/assert"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetGitCli(t *testing.T) {
	s := runtime.NewScheme()
	utilruntime.Must(cdv1.AddToScheme(s))

	app := &cdv1.Application{}
	app.Spec = cdv1.ApplicationSpec{
		Source: cdv1.ApplicationSource{
			RepoURL:        "https://github.com/tmax-cloud/cd-example-apps",
			Path:           "guestbook/guestbook-ui-svc.yaml",
			TargetRevision: "main",
		},
	}
	app.Status.Secrets = "kkkkkkkkkkkkkkkkkkkkkkkk"

	fakeCli := fake.NewFakeClientWithScheme(s, app)

	/* result*/
	_, err := GetGitCli(app, fakeCli)

	assert.Equal(t, err, nil)
	// TODO
	// - git.Client 인터페이스에 해당 필드들이 없어서 각각의 필드 개별 비교 못함
	// 방법 생각해 볼 것

	/*
		assert.Equal(t, result, github.Client{
			GitAPIURL:        cdv1.GithubDefaultAPIUrl,
			GitRepository:    "tmax-cloud/cd-example-apps",
			GitToken:         "",
			GitWebhookSecret: "kkkkkkkkkkkkkkkkkkkkkkkk",
			K8sClient:        fakeCli})
	*/
}
