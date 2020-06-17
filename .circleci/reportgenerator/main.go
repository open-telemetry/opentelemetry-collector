// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	githubAPIBaseURL = "https://api.github.com"

	// Keys of required environment variables
	projectUsernameKey = "CIRCLE_PROJECT_USERNAME"
	projectRepoNameKey = "CIRCLE_PROJECT_REPONAME"
	circleBuildURLKey  = "CIRCLE_BUILD_URL"
	jobNameKey         = "CIRCLE_JOB"
	githubAPITokenKey  = "GITHUB_TOKEN"
)

func main() {
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,
		},
	}
	logger, _ := cfg.Build()

	rg := &reportGenerator{
		logger: logger,
		httpClient: http.Client{
			Timeout: 5 * time.Second,
		},
	}

	if err := rg.getRequiredEnv(); err != nil {
		logger.Error("Required environment variable not set", zap.Error(err))
		os.Exit(1)
	}

	rg.setupDefaultHTTPHeaders()

	// Look for existing open GitHub Issue that resulted from previous
	// failures of this job.
	if err := rg.getExistingIssue(); err != nil {
		logger.Error("Failed to get exiting GitHub Issue", zap.Error(err))
		os.Exit(1)
	}

	rg.templateHelper = func(param string) string {
		switch param {
		case "jobName":
			return "`" + rg.envVariables[jobNameKey] + "`"
		case "linkToBuild":
			return os.Getenv(circleBuildURLKey)
		default:
			return ""
		}
	}

	if rg.issue == nil {
		// If none exists, create a new GitHub Issue for the failure.
		logger.Info("No existing Issues found, creating a new one.")
		if err := rg.createIssue(); err != nil {
			logger.Error("Failed to create GitHub Issue", zap.Error(err))
			os.Exit(1)
		}
	} else {
		// Otherwise, add a comment to the existing Issue.
		logger.Info("Updating GitHub Issue with latest failure")
		if err := rg.commentOnIssue(); err != nil {
			logger.Error("Failed to comment on GitHub Issue", zap.Error(err))
			os.Exit(1)
		}
	}

}

type reportGenerator struct {
	logger         *zap.Logger
	httpClient     http.Client
	envVariables   map[string]string
	headers        map[string]string
	issue          *githubIssue
	templateHelper func(string) string
}

// getRequiredEnv loads required environment variables for the main method.
// Some of the environment variables are built-in in CircleCI, whereas others
// need to be configured. See https://circleci.com/docs/2.0/env-vars/#built-in-environment-variables
// for a list of built-in environment variables.
func (rg *reportGenerator) getRequiredEnv() error {
	env := map[string]string{}

	env[projectUsernameKey] = os.Getenv(projectUsernameKey)
	env[projectRepoNameKey] = os.Getenv(projectRepoNameKey)
	env[jobNameKey] = os.Getenv(jobNameKey)
	env[githubAPITokenKey] = os.Getenv(githubAPITokenKey)

	for k, v := range env {
		if v == "" {
			return fmt.Errorf("%s environment variable not set", k)
		}
	}
	rg.envVariables = env

	return nil
}

func (rg *reportGenerator) setupDefaultHTTPHeaders() {
	headers := map[string]string{}

	headers["Authorization"] = "token " + rg.envVariables[githubAPITokenKey]
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/vnd.github.v3+json"

	rg.headers = headers
}

const (
	issueTitleTemplate = `Bug report for failed CircleCI build (job: ${jobName})`
	issueBodyTemplate  = `
Auto-generated report for ${jobName} job build.

Link to failed build: ${linkToBuild}.

Note: Information about any subsequent build failures that happen while
this issue is open, will be added as comments with more information to this issue.
`
	issueCommentTemplate = `
Link to latest failed build: ${linkToBuild}
`
)

// getExistingIssues gathers an existing GitHub Issue related to previous failures
// of the same job.
func (rg *reportGenerator) getExistingIssue() error {
	url := githubAPIBaseURL + "/search/issues"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare GET request: %v", err)
	}

	for k, v := range rg.headers {
		req.Header.Set(k, v)
	}

	q := req.URL.Query()
	q.Add("q",
		fmt.Sprintf(
			"%s in:title is:open repo:%s/%s",
			rg.getIssueTitle(),
			rg.envVariables[projectUsernameKey],
			rg.envVariables[projectRepoNameKey],
		),
	)
	req.URL.RawQuery = q.Encode()

	rg.logger.Info("Searching GitHub for existing Issues")
	resp, err := rg.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get exiting GitHub issues: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read exiting GitHub issues: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		rg.logger.Error(
			"unexpected response from GitHub",
			zap.String("status_code", string(resp.StatusCode)),
			zap.String("response", string(body)),
			zap.String("url", url),
		)
		return fmt.Errorf("unexpected response from GitHub")
	}

	var respUnmarshalled githubIssueSearchResult
	if err = json.Unmarshal(body, &respUnmarshalled); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	switch respUnmarshalled.Count {
	case 0:
		// Nothing to do if there aren't any existing issues for the job.
	case 1:
		rg.issue = &respUnmarshalled.Issues[0]
	default:
		// If there are multiple issues matching the query, pick the one
		// where there's an exact match in the Issue title. Unfortunately,
		// GitHub API does not do an exact match.
		rg.logger.Info(
			"Received multiple issues. Looking for exact match",
			zap.Int("num_issues", respUnmarshalled.Count),
		)

		requiredTitle := rg.getIssueTitle()
		for _, issue := range respUnmarshalled.Issues {
			if issue.Title == requiredTitle {
				rg.issue = &issue
				rg.logger.Info("Exact match found for Issue",
					zap.String(
						"html_url",
						issue.HTMLURL,
					),
				)
				return nil
			}
		}

		rg.logger.Info("No exact match found for Issue",
			zap.String(
				"required_title",
				requiredTitle,
			),
		)
	}

	return nil
}

// commentOnIssue adds a new comment on an existing GitHub issue with
// information about the latest failure. This method is expected to be
// called only if there's an existing open Issue for the current job.
func (rg *reportGenerator) commentOnIssue() error {
	url := fmt.Sprintf(
		"%s/repos/%s/%s/issues/%d/comments",
		githubAPIBaseURL,
		rg.envVariables[projectUsernameKey],
		rg.envVariables[projectRepoNameKey],
		rg.issue.Number,
	)

	// TODO: Extend comments to list out failed tests
	j, err := json.Marshal(map[string]interface{}{
		"body": os.Expand(issueCommentTemplate, rg.templateHelper),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal Issue comment: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return fmt.Errorf("failed to prepare POST request: %v", err)
	}

	for k, v := range rg.headers {
		req.Header.Set(k, v)
	}

	resp, err := rg.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to comment on GitHub issue: %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to comment on GitHub Issue: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		rg.logger.Error(
			"unexpected response from GitHub",
			zap.String("status_code", string(resp.StatusCode)),
			zap.String("response", string(body)),
			zap.String("url", url),
		)
		return fmt.Errorf("unexpected response from GitHub")
	}

	rg.logger.Info(
		"GitHub Issue updated",
		zap.String(
			"html_link",
			rg.issue.HTMLURL,
		),
	)
	return nil
}

// createIssue creates a new GitHub Issue corresponding to a build failure.
func (rg *reportGenerator) createIssue() error {
	url := fmt.Sprintf(
		"%s/repos/%s/%s/issues",
		githubAPIBaseURL,
		rg.envVariables[projectUsernameKey],
		rg.envVariables[projectRepoNameKey],
	)

	// TODO: Add code owners to assignees and ensure permission to set labels exist.
	j, err := json.Marshal(map[string]interface{}{
		"title":  rg.getIssueTitle(),
		"body":   os.Expand(issueBodyTemplate, rg.templateHelper),
		"labels": []string{"ci_failure"},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal Issue body: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(j))
	if err != nil {
		return fmt.Errorf("failed to prepare POST request: %v", err)
	}

	for k, v := range rg.headers {
		req.Header.Set(k, v)
	}

	resp, err := rg.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create GitHub Issue: %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to load created GitHub Issue: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		rg.logger.Error(
			"unexpected response from GitHub",
			zap.String("status_code", string(resp.StatusCode)),
			zap.String("response", string(body)),
			zap.String("url", url),
		)
		return fmt.Errorf("unexpected response from GitHub")
	}

	var githubIssueUnmarshalled githubIssue
	if err = json.Unmarshal(body, &githubIssueUnmarshalled); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}
	rg.issue = &githubIssueUnmarshalled

	rg.logger.Info(
		"Created GitHub Issue",
		zap.String(
			"html_link",
			rg.issue.HTMLURL,
		),
	)
	return nil
}

func (rg reportGenerator) getIssueTitle() string {
	return strings.Replace(issueTitleTemplate, "${jobName}", rg.envVariables[jobNameKey], 1)
}

type githubIssueSearchResult struct {
	Count  int           `json:"total_count"`
	Issues []githubIssue `json:"items"`
}

// githubIssue maps to an Issue object returned by the API.
type githubIssue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
}
