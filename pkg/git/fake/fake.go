/*
 Copyright 2021 The CI/CD Operator Authors

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package fake

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"

	"github.com/tmax-cloud/cd-operator/pkg/git"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Store as global variables - only for testing! test data should be able to be set from the outside
var (
	Users    map[string]*git.User
	Repos    map[string]*Repo
	Branches map[string]*git.Branch
)

// Repo is a repository storage
type Repo struct {
	Webhooks     map[int]*git.WebhookEntry
	UserCanWrite map[string]bool

	PullRequests       map[int]*git.PullRequest
	PullRequestDiffs   map[int]*git.Diff
	PullRequestCommits map[int][]git.Commit
	Commits            map[string][]git.Commit
	CommitStatuses     map[string][]git.CommitStatus
	Comments           map[int][]git.IssueComment
}

// Client is a gitlab client struct
type Client struct {
	Repository string
	K8sClient  client.Client
}

// Init initiates the Client
func (c *Client) Init() error {
	return nil
}

// ParseWebhook parses a webhook body for github
func (c *Client) ParseWebhook(_ http.Header, _ []byte) (*git.Webhook, error) {
	return nil, nil
}

// ListWebhook lists registered webhooks
func (c *Client) ListWebhook() ([]git.WebhookEntry, error) {
	if Repos == nil {
		return nil, fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return nil, fmt.Errorf("404 no such repository")
	}

	var res []git.WebhookEntry
	for _, w := range repo.Webhooks {
		res = append(res, *w)
	}
	return res, nil
}

// RegisterWebhook registers our webhook server to the remote git server
func (c *Client) RegisterWebhook(url string) error {
	if Repos == nil {
		return fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return fmt.Errorf("404 no such repository")
	}

	if repo.Webhooks == nil {
		return fmt.Errorf("webhooks not initialized")
	}

	id := rand.Intn(100)
	repo.Webhooks[id] = &git.WebhookEntry{ID: id, URL: url}
	return nil
}

// DeleteWebhook deletes registered webhook
func (c *Client) DeleteWebhook(id int) error {
	if Repos == nil {
		return fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return fmt.Errorf("404 no such repository")
	}

	delete(repo.Webhooks, id)
	return nil
}

// ListCommitStatuses lists commit status of the specific commit
func (c *Client) ListCommitStatuses(ref string) ([]git.CommitStatus, error) {
	if Repos == nil {
		return nil, fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return nil, fmt.Errorf("404 no such repository")
	}

	if repo.CommitStatuses == nil {
		return nil, fmt.Errorf("commit statuses not initialized")
	}

	statuses, exist := repo.CommitStatuses[ref]
	if !exist {
		return nil, fmt.Errorf("404 no such ref")
	}
	return statuses, nil
}

// SetCommitStatus sets commit status for the specific commit
func (c *Client) SetCommitStatus(sha string, status git.CommitStatus) error {
	if Repos == nil {
		return fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return fmt.Errorf("404 no such repository")
	}

	if repo.CommitStatuses == nil {
		return fmt.Errorf("commit statuses not initialized")
	}

	repo.CommitStatuses[sha] = append(repo.CommitStatuses[sha], status)
	return nil
}

// GetUserInfo gets a user's information
func (c *Client) GetUserInfo(userName string) (*git.User, error) {
	if Users == nil {
		return nil, fmt.Errorf("users not initialized")
	}
	u, exist := Users[userName]
	if !exist {
		return nil, fmt.Errorf("404 no such user")
	}
	return u, nil
}

// CanUserWriteToRepo decides if the user has write permission on the repo
func (c *Client) CanUserWriteToRepo(user git.User) (bool, error) {
	if Repos == nil {
		return false, fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return false, fmt.Errorf("404 no such repository")
	}

	if repo.UserCanWrite == nil {
		return false, fmt.Errorf("userCanWrite not initialized")
	}

	privilege, exist := repo.UserCanWrite[user.Name]
	if !exist {
		return false, fmt.Errorf("404 no such user")
	}

	return privilege, nil
}

// RegisterComment registers comment to an issue
func (c *Client) RegisterComment(_ git.IssueType, issueNo int, body string) error {
	if Repos == nil {
		return fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return fmt.Errorf("404 no such repository")
	}

	if repo.Comments == nil {
		return fmt.Errorf("comments not initialized")
	}

	t := metav1.Now()
	repo.Comments[issueNo] = append(repo.Comments[issueNo], git.IssueComment{
		Comment: git.Comment{Body: body, CreatedAt: &t},
		Issue: git.Issue{
			PullRequest: &git.PullRequest{
				ID: issueNo,
			},
		},
	})
	return nil
}

// ListPullRequests gets pull request list
func (c *Client) ListPullRequests(_ bool) ([]git.PullRequest, error) {
	if Repos == nil {
		return nil, fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return nil, fmt.Errorf("404 no such repository")
	}

	var prs []git.PullRequest
	for _, pr := range repo.PullRequests {
		prs = append(prs, *pr)
	}

	return prs, nil
}

// GetPullRequest gets PR given id
func (c *Client) GetPullRequest(id int) (*git.PullRequest, error) {
	if Repos == nil {
		return nil, fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return nil, fmt.Errorf("404 no such repository")
	}

	if repo.PullRequests == nil {
		return nil, fmt.Errorf("pull requests not initialized")
	}

	pr, exist := repo.PullRequests[id]
	if !exist {
		return nil, fmt.Errorf("404 no such pr")
	}

	return pr, nil
}

// MergePullRequest merges a pull request
func (c *Client) MergePullRequest(id int, _ string, _ git.MergeMethod, message string) error {
	if Repos == nil {
		return fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return fmt.Errorf("404 no such repository")
	}

	pr, exist := repo.PullRequests[id]
	if !exist {
		return fmt.Errorf("404 no such pr")
	}

	repo.PullRequests[id].Mergeable = false
	repo.PullRequests[id].State = git.PullRequestStateClosed
	commit := git.Commit{
		SHA:     pr.Head.Sha,
		Message: message,
	}

	if message == "" {
		commit.Message = fmt.Sprintf("%s(#%d)", pr.Title, pr.ID)
	}

	repo.Commits[pr.Base.Ref] = append(repo.Commits[pr.Base.Ref], commit)

	return nil
}

// GetPullRequestDiff gets diff of the pull request
func (c *Client) GetPullRequestDiff(id int) (*git.Diff, error) {
	if Repos == nil {
		return nil, fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return nil, fmt.Errorf("404 no such repository")
	}

	if repo.PullRequestDiffs == nil {
		return nil, fmt.Errorf("pull request diffs not initialized")
	}

	diff, exist := repo.PullRequestDiffs[id]
	if !exist {
		return nil, fmt.Errorf("404 no such pr")
	}

	return diff, nil
}

// ListPullRequestCommits lists commits list of a pull request
func (c *Client) ListPullRequestCommits(id int) ([]git.Commit, error) {
	if Repos == nil {
		return nil, fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return nil, fmt.Errorf("404 no such repository")
	}

	if repo.PullRequestCommits == nil {
		return nil, fmt.Errorf("pull request commits not initialized")
	}

	commits, exist := repo.PullRequestCommits[id]
	if !exist {
		return nil, fmt.Errorf("404 no such pr")
	}

	return commits, nil
}

// SetLabel sets label to the issue id
func (c *Client) SetLabel(_ git.IssueType, id int, label string) error {
	if Repos == nil {
		return fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[c.Repository]
	if !repoExist {
		return fmt.Errorf("404 no such repository")
	}

	if repo.PullRequests == nil {
		return fmt.Errorf("pull requests not initialized")
	}

	pr, exist := repo.PullRequests[id]
	if !exist {
		return fmt.Errorf("404 no such PR")
	}

	pr.Labels = append(pr.Labels, git.IssueLabel{Name: label})

	return nil
}

// DeleteLabel deletes label from the issue id
func (c *Client) DeleteLabel(_ git.IssueType, id int, label string) error {
	return DeleteLabel(c.Repository, id, label)
}

// GetBranch returns branch info
func (c *Client) GetBranch(branch string) (*git.Branch, error) {
	if Branches == nil {
		return nil, fmt.Errorf("branches not initialized")
	}
	b, exist := Branches[branch]
	if !exist {
		return nil, fmt.Errorf("404 no such branch (%s)", branch)
	}
	return b, nil
}

// DeleteLabel deletes label from a pull request
func DeleteLabel(repoName string, id int, label string) error {
	if Repos == nil {
		return fmt.Errorf("repos not initialized")
	}
	repo, repoExist := Repos[repoName]
	if !repoExist {
		return fmt.Errorf("404 no such repository")
	}

	if repo.PullRequests == nil {
		return fmt.Errorf("pull requests not initialized")
	}

	pr, exist := repo.PullRequests[id]
	if !exist {
		return fmt.Errorf("404 no such PR")
	}

	idx := -1
	for i, l := range pr.Labels {
		if l.Name == label {
			idx = i
			break
		}
	}
	if idx >= 0 {
		if idx == len(pr.Labels)-1 {
			pr.Labels = pr.Labels[:idx]
		} else {
			pr.Labels = append(pr.Labels[:idx], pr.Labels[idx+1:]...)
		}
	}

	return nil
}

// GetManifestURLs gets manifests' URLs
func (c *Client) GetManifestURLs(path, revision string) ([]git.DownloadURL, error) {
	var downloadURLs []git.DownloadURL
	var downloadURL git.DownloadURL

	testBody := map[string]string{
		"guestbook":       `[{"name":"guestbook-test-svc.yaml","path":"guestbook/guestbook-test-svc.yaml","sha":"e8a4a27fbae4042ba3428098c0b899f3665c39e4","size":141,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-test-svc.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-test-svc.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/e8a4a27fbae4042ba3428098c0b899f3665c39e4","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-test-svc.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-test-svc.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/e8a4a27fbae4042ba3428098c0b899f3665c39e4","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-test-svc.yaml"}},{"name":"guestbook-ui-deployment.yaml","path":"guestbook/guestbook-ui-deployment.yaml","sha":"8a0975e363539eacfba296559ad6385cbedd1245","size":389,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-deployment.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-deployment.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/8a0975e363539eacfba296559ad6385cbedd1245","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-deployment.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-deployment.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/8a0975e363539eacfba296559ad6385cbedd1245","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-deployment.yaml"}},{"name":"guestbook-ui-svc.yaml","path":"guestbook/guestbook-ui-svc.yaml","sha":"fa173a2b2e84c2a3566a1572bbc65a72155b58d1","size":145,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-svc.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/fa173a2b2e84c2a3566a1572bbc65a72155b58d1","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/fa173a2b2e84c2a3566a1572bbc65a72155b58d1","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-svc.yaml"}},{"name":"test","path":"guestbook/test","sha":"7eb2aed0d0aadb4fd268b7e7921e9eb9c61d2a1e","size":0,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/tree/main/guestbook/test","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/trees/7eb2aed0d0aadb4fd268b7e7921e9eb9c61d2a1e","download_url":null,"type":"dir","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/trees/7eb2aed0d0aadb4fd268b7e7921e9eb9c61d2a1e","html":"https://github.com/tmax-cloud/cd-example-apps/tree/main/guestbook/test"}}]`,
		"guestbook/test":  `[{"name":"guestbook-testui-deployment.yaml","path":"guestbook/test/guestbook-testui-deployment.yaml","sha":"28322ec77cc65392aee4a6ea312a7a8e67e04a71","size":399,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test/guestbook-testui-deployment.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/test/guestbook-testui-deployment.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/28322ec77cc65392aee4a6ea312a7a8e67e04a71","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/test/guestbook-testui-deployment.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test/guestbook-testui-deployment.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/28322ec77cc65392aee4a6ea312a7a8e67e04a71","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/test/guestbook-testui-deployment.yaml"}}]`,
		"deployment.yaml": `{"name":"deployment.yaml","path":"deployment.yaml","sha":"2d0f44780d8fe8108524a77f96d10da2231e1e90","size":345,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/deployment.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/deployment.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/2d0f44780d8fe8108524a77f96d10da2231e1e90","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/deployment.yaml","type":"file","content":"YXBpVmVyc2lvbjogYXBwcy92MQpraW5kOiBEZXBsb3ltZW50Cm1ldGFkYXRh\nOgogIG5hbWU6IHRlc3QtZGVwbG95LWZyb20tZ2l0CnNwZWM6CiAgdGVtcGxh\ndGU6CiAgICBtZXRhZGF0YToKICAgICAgbmFtZTogbmdpbngKICAgICAgbGFi\nZWxzOgogICAgICAgIGFwcHM6IHRlc3QtYXBwCiAgICBzcGVjOgogICAgICBj\nb250YWluZXJzOgogICAgICAgIC0gbmFtZTogbmdpbngtY29udGFpbmVyCiAg\nICAgICAgICBpbWFnZTogbmdpbngKICAgICAgICAgIHBvcnRzOgogICAgICAg\nICAgICAtIGNvbnRhaW5lclBvcnQ6IDgwCiAgc2VsZWN0b3I6CiAgICBtYXRj\naExhYmVsczoKICAgICAgYXBwczogdGVzdC1hcHAK\n","encoding":"base64","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/deployment.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/2d0f44780d8fe8108524a77f96d10da2231e1e90","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/deployment.yaml"}}`,
	}

	if path == "invalid" {
		return nil, fmt.Errorf("404 not found")
	}

	body := io.NopCloser(strings.NewReader(testBody[path]))

	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(raw, &downloadURLs); err != nil {
		if err := json.Unmarshal(raw, &downloadURL); err != nil {
			return nil, err
		}
		downloadURLs = append(downloadURLs, downloadURL)
	}
	return downloadURLs, nil
}
