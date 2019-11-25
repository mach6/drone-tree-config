package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bitsbeats/drone-tree-config/plugin/scm_clients"
	"github.com/drone/drone-go/drone"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NewScmClient creates a new client for the git provider
func (p *Plugin) NewScmClient(uuid uuid.UUID, repo drone.Repo, ctx context.Context) scm_clients.ScmClient {
	var scmClient scm_clients.ScmClient
	var err error
	if p.gitHubToken != "" {
		scmClient, err = scm_clients.NewGitHubClient(uuid, p.server, p.gitHubToken, repo, ctx)
	} else if p.bitBucketClient != "" {
		scmClient, err = scm_clients.NewBitBucketClient(uuid, p.authServer, p.server, p.bitBucketClient, p.bitBucketSecret, repo)
	} else {
		err = fmt.Errorf("no SCM credentials specified")
	}
	if err != nil {
		logrus.Errorf("Unable to connect to SCM server.")
	}
	return scmClient
}

// getChanges tries to get a list of changed files from github
func (p *Plugin) getScmChanges(ctx context.Context, req *request) ([]string, error) {
	var changedFiles []string

	if req.Build.Trigger == "@cron" {
		// cron jobs trigger a full build
		changedFiles = []string{}
	} else if strings.HasPrefix(req.Build.Ref, "refs/pull/") {
		// use pullrequests api to get changed files
		pullRequestID, err := strconv.Atoi(strings.Split(req.Build.Ref, "/")[2])
		if err != nil {
			logrus.Errorf("%s unable to get pull request id %v", req.UUID, err)
			return nil, err
		}
		changedFiles, err = req.Client.ChangedFilesInPullRequest(ctx, pullRequestID)
		if err != nil {
			logrus.Errorf("%s unable to fetch diff for Pull request %v", req.UUID, err)
		}
	} else {
		// use diff to get changed files
		before := req.Build.Before
		if before == "0000000000000000000000000000000000000000" || before == "" {
			before = fmt.Sprintf("%s~1", req.Build.After)
		}
		var err error
		changedFiles, err = req.Client.ChangedFilesInDiff(ctx, before, req.Build.After)
		if err != nil {
			logrus.Errorf("%s unable to fetch diff: '%v'", req.UUID, err)
			return nil, err
		}
	}

	if len(changedFiles) > 0 {
		changedList := strings.Join(changedFiles, "\n  ")
		logrus.Debugf("%s changed files: \n  %s", req.UUID, changedList)
	} else {
		return nil, nil
	}
	return changedFiles, nil
}

// getFile downloads a file from github
func (p *Plugin) getScmFile(ctx context.Context, req *request, file string) (content string, err error) {
	logrus.Debugf("%s checking %s/%s %s", req.UUID, req.Repo.Namespace, req.Repo.Name, file)
	return req.Client.GetFileContents(ctx, file, req.Build.After)
}
