package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FileExists checks if the file exists in path
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

/*
// GetGitCli generates git client, depending on the git type in the cfg
func GetGitCli(cfg *cdv1.Application, cli client.Client) (git.Client, error) {
	var c git.Client
	switch cfg.Spec.Git.Type {
	case cdv1.GitTypeGitHub:
		c = &github.Client{Application: cfg, K8sClient: cli}
	case cdv1.GitTypeGitLab:
		c = &gitlab.Client{Application: cfg, K8sClient: cli}
	case cdv1.GitTypeFake:
		c = &fake.Client{Application: cfg, K8sClient: cli}
	default:
		return nil, fmt.Errorf("git type %s is not supported", cfg.Spec.Git.Type)
	}
	if err := c.Init(); err != nil {
		return nil, err
	}
	return c, nil
}
*/

// ParseApproversList parses user/email from line-separated and comma-separated approvers list
func ParseApproversList(str string) ([]string, error) {
	var approvers []string

	// Regexp for verifying if it's in form
	re := regexp.MustCompile("[^=]+(=.+)?")

	lineSep := strings.Split(strings.TrimSpace(str), "\n")
	for _, line := range lineSep {
		commaSep := strings.Split(strings.TrimSpace(line), ",")
		for _, approver := range commaSep {
			trimmed := strings.TrimSpace(approver)
			if re.MatchString(trimmed) {
				approvers = append(approvers, trimmed)
			} else {
				return nil, fmt.Errorf("comma-separated approver %s is not in form of <user-name>[=<email>](optional)", approver)
			}
		}
	}

	return approvers, nil
}

// ParseEmailFromUsers parses email from approvers list
func ParseEmailFromUsers(users []string) []string {
	var emails []string

	emailRe := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	for _, u := range users {
		subs := strings.Split(u, "=")
		if len(subs) < 2 {
			continue
		}
		trimmed := strings.TrimSpace(subs[1])
		if emailRe.MatchString(trimmed) {
			emails = append(emails, trimmed)
		}
	}

	return emails
}
