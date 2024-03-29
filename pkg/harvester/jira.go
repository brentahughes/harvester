package harvester

import (
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

const issueQuery = `assignee = currentUser() AND Resolution = Unresolved AND status not in ("To Do", "Selected")`

func (h *harvester) getNewJiraClient() error {
	tp := jira.BasicAuthTransport{
		Username: h.Settings.Jira.User,
		Password: h.Settings.Jira.Pass,
		Transport: &http.Transport{DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		},
	}

	var err error
	h.jiraClient, err = jira.NewClient(tp.Client(), h.Settings.Jira.URL)
	return err
}

func (h *harvester) getUsersActiveIssues() ([]jira.Issue, error) {
	issues, _, err := h.jiraClient.Issue.Search(issueQuery, nil)
	if err != nil {
		log.Print(err)
		return nil, errors.New("error getting active jira issues")
	}
	return issues, err
}

func (h *harvester) getJiraByKey(key string) (*jira.Issue, error) {
	issue, _, err := h.jiraClient.Issue.Get(key, nil)
	return issue, err
}

func (h *harvester) harvestToJira(task *harvestTask) jira.Issue {
	return jira.Issue{
		Key: *task.Project.Code,
		Fields: &jira.IssueFields{
			Summary: *task.Project.Name,
		},
	}
}
