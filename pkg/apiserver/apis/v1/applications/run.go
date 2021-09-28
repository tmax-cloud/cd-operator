package applications

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/apiserver"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/pkg/git"
	"github.com/tmax-cloud/cd-operator/pkg/server"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultBranch = "master"
)

func (h *handler) runPreHandler(w http.ResponseWriter, req *http.Request) {
	h.runHandler(w, req, git.EventTypePullRequest)
}

func (h *handler) runPostHandler(w http.ResponseWriter, req *http.Request) {
	h.runHandler(w, req, git.EventTypePush)
}

func (h *handler) runHandler(w http.ResponseWriter, req *http.Request, et git.EventType) {
	reqID := utils.RandomString(10)
	log := h.log.WithValues("request", reqID)

	// Get ns/resource name
	vars := mux.Vars(req)

	ns, nsExist := vars[apiserver.NamespaceParamKey]
	resName, nameExist := vars[appParamKey]
	if !nsExist || !nameExist {
		log.Info("url is malformed")
		_ = utils.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}

	// Get user
	user, err := apiserver.GetUserName(req.Header)
	if err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusUnauthorized, fmt.Sprintf("req: %s, forbidden user, err : %s", reqID, err.Error()))
		return
	}
	userEscaped := regexp.MustCompile("[^-A-Za-z0-9_.]").ReplaceAllString(user, "_")

	// Get Application
	app := &cdv1.Application{}
	if err := h.k8sClient.Get(context.Background(), types.NamespacedName{Name: resName, Namespace: ns}, app); err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot get application %s/%s", reqID, ns, resName))
		return
	}

	gitHost, err := app.Spec.Git.GetGitHost()
	if err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot get Application %s/%s's git host", reqID, ns, resName))
		return
	}

	// Build webhook
	wh := &git.Webhook{
		EventType: et,
		Repo: git.Repository{
			Name: app.Spec.Git.Repository,
			URL:  fmt.Sprintf("%s/%s", gitHost, app.Spec.Git.Repository),
		},
	}

	switch et {
	case git.EventTypePullRequest:
		pr, err := buildPullRequestWebhook(req.Body, userEscaped)
		if err != nil {
			log.Info(err.Error())
			_ = utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("req: %s, cannot build pull_request webhook", reqID))
			return
		}
		wh.PullRequest = pr
	case git.EventTypePush:
		push, err := buildPushWebhook(req.Body)
		if err != nil {
			log.Info(err.Error())
			_ = utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("req: %s, cannot build push webhook", reqID))
			return
		}
		wh.Push = push
	}
	wh.Sender = git.User{
		Name: fmt.Sprintf("trigger-%s-end", userEscaped),
	}

	// Trigger Run!
	if err := server.HandleEvent(wh, app); err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot handle event, err : %s", reqID, err.Error()))
		return
	}

	_ = utils.RespondJSON(w, struct{}{})
}

func buildPullRequestWebhook(body io.ReadCloser, user string) (*git.PullRequest, error) {
	userReq := &cdv1.ApplicationAPIReqRunPreBody{}
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(userReq); err != nil {
		return nil, err
	}

	baseBranch := userReq.BaseBranch
	headBranch := userReq.HeadBranch
	if baseBranch == "" {
		baseBranch = defaultBranch
	}
	if headBranch == "" {
		return nil, fmt.Errorf("head_branch must be set")
	}

	return &git.PullRequest{
		State:  git.PullRequestStateOpen,
		Action: git.PullRequestActionOpen,
		Author: git.User{
			Name: fmt.Sprintf("trigger-%s-end", user),
		},
		Base: git.Base{
			Ref: baseBranch,
			Sha: git.FakeSha,
		},
		Head: git.Head{
			Ref: headBranch,
			Sha: git.FakeSha,
		},
	}, nil
}

func buildPushWebhook(body io.ReadCloser) (*git.Push, error) {
	userReq := &cdv1.ApplicationAPIReqRunPostBody{}
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(userReq); err != nil {
		return nil, err
	}

	branch := userReq.Branch
	if branch == "" {
		branch = defaultBranch
	}

	return &git.Push{
		Ref: branch,
		Sha: git.FakeSha,
	}, nil
}
