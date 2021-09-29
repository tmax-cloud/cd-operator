package utils

import (
	"fmt"
	"os"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/git"
	"github.com/tmax-cloud/cd-operator/pkg/git/fake"
	"github.com/tmax-cloud/cd-operator/pkg/git/github"
	"github.com/tmax-cloud/cd-operator/pkg/git/gitlab"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FileExists checks if the file exists in path
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetGitCli generates git client, depending on the git type in the app
func GetGitCli(app *cdv1.Application, cli client.Client) (git.Client, error) {
	// TODO : Refactoring
	var c git.Client

	gitType := app.Spec.Source.GetGitType()
	apiurl := app.Spec.Source.GetAPIUrl()
	gitRepo := app.Spec.Source.GetRepository()
	gitToken := ""
	/* TODO : Webhook 등록하는 로직 구현 되면, 살릴 예정
	gitToken, err := app.GetToken(cli)
	if err != nil {
		return nil, err
	}
	*/
	// webhook parsing 할 때, validating 시 사용
	webhookSecret := app.Status.Secrets
	switch gitType {
	case cdv1.GitTypeGitHub:
		c = &github.Client{
			GitAPIURL:        apiurl,
			GitRepository:    gitRepo,
			GitToken:         gitToken,
			GitWebhookSecret: webhookSecret,
			K8sClient:        cli}
	case cdv1.GitTypeGitLab:
		c = &gitlab.Client{
			GitAPIURL:        apiurl,
			GitRepository:    gitRepo,
			GitToken:         gitToken,
			GitWebhookSecret: webhookSecret,
			K8sClient:        cli}
	case cdv1.GitTypeFake:
		c = &fake.Client{Repository: gitRepo, K8sClient: cli}
	default:
		return nil, fmt.Errorf("git type %s is not supported", gitType)
	}
	if err := c.Init(); err != nil {
		return nil, err
	}
	return c, nil
}
