// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/joshdk/go-junit"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

const (
	// Keys of required environment variables
	projectUsernameKey = "CIRCLE_PROJECT_USERNAME"
	projectRepoNameKey = "CIRCLE_PROJECT_REPONAME"
	circleBuildURLKey  = "CIRCLE_BUILD_URL"
	jobNameKey         = "CIRCLE_JOB"
	githubAPITokenKey  = "GITHUB_TOKEN" // #nosec G101
)

func main() {
	pathToArtifacts := ""
	if len(os.Args) > 1 {
		pathToArtifacts = os.Args[1]
	}

	rg := newReportGenerator(pathToArtifacts)

	// Look for existing open GitHub Issue that resulted from previous
	// failures of this job.
	rg.logger.Info("Searching GitHub for existing Issues")
	existingIssue := rg.getExistingIssue()

	if existingIssue == nil {
		// If none exists, create a new GitHub Issue for the failure.
		rg.logger.Info("No existing Issues found, creating a new one.")
		createdIssue := rg.createIssue()
		rg.logger.Info("New GitHub Issue created", zap.String("html_url", *createdIssue.HTMLURL))
	} else {
		// Otherwise, add a comment to the existing Issue.
		rg.logger.Info(
			"Updating GitHub Issue with latest failure",
			zap.String("html_url", *existingIssue.HTMLURL),
		)
		createdIssueComment := rg.commentOnIssue(existingIssue)
		rg.logger.Info("GitHub Issue updated", zap.String("html_url", *createdIssueComment.HTMLURL))
	}
}

func newReportGenerator(pathToArtifacts string) *reportGenerator {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("Failed to set up logger: %v", err)
		os.Exit(1)
	}

	rg := &reportGenerator{
		ctx:    context.Background(),
		logger: logger,
	}

	rg.getRequiredEnv()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: rg.envVariables[githubAPITokenKey]})
	tc := oauth2.NewClient(rg.ctx, ts)
	rg.client = github.NewClient(tc)

	if pathToArtifacts != "" {
		rg.logger.Info("Ingesting test reports", zap.String("path", pathToArtifacts))
		suites, err := junit.IngestFile(pathToArtifacts)
		if err != nil {
			rg.logger.Warn(
				"Failed to ingest JUnit xml, omitting test results from report",
				zap.Error(err),
			)
		}

		rg.testSuites = suites
	}

	return rg
}

type reportGenerator struct {
	ctx          context.Context
	logger       *zap.Logger
	client       *github.Client
	envVariables map[string]string
	testSuites   []junit.Suite
}

// getRequiredEnv loads required environment variables for the main method.
// Some of the environment variables are built-in in CircleCI, whereas others
// need to be configured. See https://circleci.com/docs/2.0/env-vars/#built-in-environment-variables
// for a list of built-in environment variables.
func (rg *reportGenerator) getRequiredEnv() {
	env := map[string]string{}

	env[projectUsernameKey] = os.Getenv(projectUsernameKey)
	env[projectRepoNameKey] = os.Getenv(projectRepoNameKey)
	env[jobNameKey] = os.Getenv(jobNameKey)
	env[githubAPITokenKey] = os.Getenv(githubAPITokenKey)

	for k, v := range env {
		if v == "" {
			rg.logger.Fatal(
				"Required environment variable not set",
				zap.String("env_var", k),
			)
		}
	}

	rg.envVariables = env
}

const (
	issueTitleTemplate = `Bug report for failed CircleCI build (job: ${jobName})`
	issueBodyTemplate  = `
Auto-generated report for ${jobName} job build.

Link to failed build: ${linkToBuild}

${failedTests}

**Note**: Information about any subsequent build failures that happen while
this issue is open, will be added as comments with more information to this issue.
`
	issueCommentTemplate = `
Link to latest failed build: ${linkToBuild}

${failedTests}
`
)

func (rg reportGenerator) templateHelper(param string) string {
	switch param {
	case "jobName":
		return "`" + rg.envVariables[jobNameKey] + "`"
	case "linkToBuild":
		return os.Getenv(circleBuildURLKey)
	case "failedTests":
		return rg.getFailedTests()
	default:
		return ""
	}
}

// getExistingIssues gathers an existing GitHub Issue related to previous failures
// of the same job.
func (rg *reportGenerator) getExistingIssue() *github.Issue {
	issues, response, err := rg.client.Issues.ListByRepo(
		rg.ctx,
		rg.envVariables[projectUsernameKey],
		rg.envVariables[projectRepoNameKey],
		&github.IssueListByRepoOptions{
			State: "open",
		},
	)
	if err != nil {
		rg.logger.Fatal("Failed to search GitHub Issues", zap.Error(err))
	}

	if response.StatusCode != http.StatusOK {
		rg.handleBadResponses(response)
	}

	requiredTitle := rg.getIssueTitle()
	for _, issue := range issues {
		if *issue.Title == requiredTitle {
			return issue
		}
	}

	return nil
}

// commentOnIssue adds a new comment on an existing GitHub issue with
// information about the latest failure. This method is expected to be
// called only if there's an existing open Issue for the current job.
func (rg *reportGenerator) commentOnIssue(issue *github.Issue) *github.IssueComment {
	body := os.Expand(issueCommentTemplate, rg.templateHelper)

	issueComment, response, err := rg.client.Issues.CreateComment(
		rg.ctx,
		rg.envVariables[projectUsernameKey],
		rg.envVariables[projectRepoNameKey],
		*issue.Number,
		&github.IssueComment{
			Body: &body,
		},
	)
	if err != nil {
		rg.logger.Fatal("Failed to search GitHub Issues", zap.Error(err))
	}

	if response.StatusCode != http.StatusCreated {
		rg.handleBadResponses(response)
	}

	return issueComment
}

// createIssue creates a new GitHub Issue corresponding to a build failure.
func (rg *reportGenerator) createIssue() *github.Issue {
	title := rg.getIssueTitle()
	body := os.Expand(issueBodyTemplate, rg.templateHelper)

	issue, response, err := rg.client.Issues.Create(
		rg.ctx,
		rg.envVariables[projectUsernameKey],
		rg.envVariables[projectRepoNameKey],
		&github.IssueRequest{
			Title: &title,
			Body:  &body,
			// TODO: Set Assignees and labels
		})
	if err != nil {
		rg.logger.Fatal("Failed to create GitHub Issue", zap.Error(err))
	}

	if response.StatusCode != http.StatusCreated {
		rg.handleBadResponses(response)
	}

	return issue
}

func (rg reportGenerator) getIssueTitle() string {
	return strings.Replace(issueTitleTemplate, "${jobName}", rg.envVariables[jobNameKey], 1)
}

// getFailedTests returns information about failed tests if available, otherwise
// an empty string.
func (rg reportGenerator) getFailedTests() string {
	if len(rg.testSuites) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("#### Test Failures\n")

	for _, s := range rg.testSuites {
		for _, t := range s.Tests {
			if t.Status != junit.StatusFailed {
				continue
			}
			sb.WriteString("-  " + t.Name + "\n")
		}
	}

	return sb.String()
}

func (rg reportGenerator) handleBadResponses(response *github.Response) {
	body, _ := ioutil.ReadAll(response.Body)
	rg.logger.Fatal(
		"Unexpected response from GitHub",
		zap.Int("status_code", response.StatusCode),
		zap.String("response", string(body)),
		zap.String("url", response.Request.URL.String()),
	)
}
