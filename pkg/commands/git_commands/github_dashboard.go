package git_commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/config"
)

var dashboardCacheMutex sync.Mutex

func dashboardCacheFilePath() string {
	path, err := config.StateFilePath("github_dashboard.json")
	if err != nil {
		return ""
	}
	return path
}

func LoadCachedDashboard(repoPath string) *GithubDashboardData {
	dashboardCacheMutex.Lock()
	defer dashboardCacheMutex.Unlock()

	content, err := os.ReadFile(dashboardCacheFilePath())
	if err != nil || len(content) == 0 {
		return nil
	}
	var byRepo map[string]*GithubDashboardData
	if err := json.Unmarshal(content, &byRepo); err != nil {
		return nil
	}
	return byRepo[repoPath]
}

func SaveCachedDashboard(repoPath string, data *GithubDashboardData) {
	if data == nil {
		return
	}
	dashboardCacheMutex.Lock()
	defer dashboardCacheMutex.Unlock()

	path := dashboardCacheFilePath()
	byRepo := map[string]*GithubDashboardData{}
	if content, err := os.ReadFile(path); err == nil && len(content) > 0 {
		_ = json.Unmarshal(content, &byRepo)
	}
	byRepo[repoPath] = data
	if encoded, err := json.Marshal(byRepo); err == nil {
		_ = os.WriteFile(path, encoded, 0o600)
	}
}

type GithubDashboardData struct {
	PullRequests []*models.GithubPullRequest
	Issues       []*models.GithubIssue
	ProjectItems []*models.GithubProjectItem
}

type dashboardGraphQLResponse struct {
	Data struct {
		Repository struct {
			PullRequests struct {
				Nodes []dashboardPRNode `json:"nodes"`
			} `json:"pullRequests"`
			Issues struct {
				Nodes []dashboardIssueNode `json:"nodes"`
			} `json:"issues"`
			ProjectsV2 struct {
				Nodes []dashboardProjectNode `json:"nodes"`
			} `json:"projectsV2"`
		} `json:"repository"`
		RepositoryOwner struct {
			ProjectsV2 struct {
				Nodes []dashboardProjectNode `json:"nodes"`
			} `json:"projectsV2"`
		} `json:"repositoryOwner"`
		Viewer struct {
			ProjectsV2 struct {
				Nodes []dashboardProjectNode `json:"nodes"`
			} `json:"projectsV2"`
		} `json:"viewer"`
	} `json:"data"`
}

type dashboardPRNode struct {
	Number              int    `json:"number"`
	Title               string `json:"title"`
	State               string `json:"state"`
	IsDraft             bool   `json:"isDraft"`
	Url                 string `json:"url"`
	HeadRefName         string `json:"headRefName"`
	BaseRefName         string `json:"baseRefName"`
	Body                string `json:"body"`
	UpdatedAt           string `json:"updatedAt"`
	Author              struct {
		Login string `json:"login"`
	} `json:"author"`
	HeadRepositoryOwner struct {
		Login string `json:"login"`
	} `json:"headRepositoryOwner"`
	HeadRef struct {
		Target struct {
			StatusCheckRollup struct {
				State string `json:"state"`
			} `json:"statusCheckRollup"`
		} `json:"target"`
	} `json:"headRef"`
	Labels struct {
		Nodes []models.GithubLabel `json:"nodes"`
	} `json:"labels"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Comments struct {
		Nodes []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
		} `json:"nodes"`
	} `json:"comments"`
}

type dashboardIssueNode struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Url       string `json:"url"`
	Body      string `json:"body"`
	UpdatedAt string `json:"updatedAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Labels struct {
		Nodes []models.GithubLabel `json:"nodes"`
	} `json:"labels"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Comments struct {
		Nodes []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
		} `json:"nodes"`
	} `json:"comments"`
}

type dashboardProjectNode struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Url    string `json:"url"`
	Fields struct {
		Nodes []struct {
			ID      string                            `json:"id"`
			Name    string                            `json:"name"`
			Options []models.GithubProjectStatusOption `json:"options"`
		} `json:"nodes"`
	} `json:"fields"`
	Items struct {
		Nodes []struct {
			ID               string `json:"id"`
			FieldValueByName *struct {
				Name     string `json:"name"`
				OptionID string `json:"optionId"`
			} `json:"fieldValueByName"`
			Content struct {
				Typename   string `json:"__typename"`
				Number     int    `json:"number"`
				Title      string `json:"title"`
				State      string `json:"state"`
				Url        string `json:"url"`
				Body       string `json:"body"`
				Repository struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"repository"`
				Author struct {
					Login string `json:"login"`
				} `json:"author"`
				Labels struct {
					Nodes []models.GithubLabel `json:"nodes"`
				} `json:"labels"`
			} `json:"content"`
		} `json:"nodes"`
	} `json:"items"`
}

const dashboardGraphQLQuery = `
query($owner: String!, $repo: String!) {
  repository(owner: $owner, name: $repo) {
    pullRequests(first: 25, states: [OPEN], orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        number
        title
        state
        isDraft
        url
        headRefName
        baseRefName
        body
        updatedAt
        author { login }
        headRepositoryOwner { login }
        labels(first: 6) { nodes { name color } }
        assignees(first: 3) { nodes { login } }
        comments(last: 5) { nodes { author { login } body createdAt } }
      }
    }
    issues(first: 25, states: [OPEN], orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        number
        title
        state
        url
        body
        updatedAt
        author { login }
        labels(first: 6) { nodes { name color } }
        assignees(first: 3) { nodes { login } }
        comments(last: 5) { nodes { author { login } body createdAt } }
      }
    }
    projectsV2(first: 5) {
      nodes {
        id
        number
        title
        url
        fields(first: 15) {
          nodes {
            ... on ProjectV2SingleSelectField {
              id
              name
              options { id name }
            }
          }
        }
        items(first: 25) {
          nodes {
            id
            fieldValueByName(name: "Status") {
              ... on ProjectV2ItemFieldSingleSelectValue {
                name
                optionId
              }
            }
            content {
              __typename
              ... on Issue {
                number
                title
                state
                url
                body
                repository { nameWithOwner }
                author { login }
                labels(first: 5) { nodes { name color } }
              }
              ... on PullRequest {
                number
                title
                state
                url
                body
                repository { nameWithOwner }
                author { login }
                labels(first: 5) { nodes { name color } }
              }
              ... on DraftIssue {
                title
                body
              }
            }
          }
        }
      }
    }
  }
  viewer {
    projectsV2(first: 5) {
      nodes {
        id
        number
        title
        url
        fields(first: 15) {
          nodes {
            ... on ProjectV2SingleSelectField {
              id
              name
              options { id name }
            }
          }
        }
        items(first: 25) {
          nodes {
            id
            fieldValueByName(name: "Status") {
              ... on ProjectV2ItemFieldSingleSelectValue {
                name
                optionId
              }
            }
            content {
              __typename
              ... on Issue {
                number
                title
                state
                url
                body
                repository { nameWithOwner }
                author { login }
                labels(first: 5) { nodes { name color } }
              }
              ... on PullRequest {
                number
                title
                state
                url
                body
                repository { nameWithOwner }
                author { login }
                labels(first: 5) { nodes { name color } }
              }
              ... on DraftIssue {
                title
                body
              }
            }
          }
        }
      }
    }
  }
}
`

func (self *GitHubCommands) FetchDashboardData(serviceInfo *hosting_service.ServiceInfo, token string) (*GithubDashboardData, error) {
	endpoint := graphQLEndpoint(serviceInfo.WebDomain)
	variables := map[string]string{
		"owner": serviceInfo.Owner,
		"repo":  serviceInfo.Repository,
	}

	bodyBytes, err := json.Marshal(graphQLRequest{
		Query:     dashboardGraphQLQuery,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyStr := new(bytes.Buffer)
		_, _ = bodyStr.ReadFrom(resp.Body)
		return nil, fmt.Errorf("GraphQL query failed with status: %s. Body: %s", resp.Status, bodyStr.String())
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseDashboardResponse(respBytes, serviceInfo.Owner+"/"+serviceInfo.Repository)
}

func parseDashboardResponse(respBytes []byte, currentRepoFullName string) (*GithubDashboardData, error) {
	var parsed dashboardGraphQLResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, err
	}

	prs := make([]*models.GithubPullRequest, 0, len(parsed.Data.Repository.PullRequests.Nodes))
	for _, n := range parsed.Data.Repository.PullRequests.Nodes {
		state := n.State
		if n.IsDraft && state != "CLOSED" {
			state = "DRAFT"
		}
		assignees := make([]string, 0, len(n.Assignees.Nodes))
		for _, a := range n.Assignees.Nodes {
			assignees = append(assignees, a.Login)
		}
		comments := make([]models.GithubComment, 0, len(n.Comments.Nodes))
		for _, c := range n.Comments.Nodes {
			comments = append(comments, models.GithubComment{
				Author:    c.Author.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt,
			})
		}
		prs = append(prs, &models.GithubPullRequest{
			Number:      n.Number,
			Title:       n.Title,
			State:       state,
			ChecksState: n.HeadRef.Target.StatusCheckRollup.State,
			Url:         n.Url,
			HeadRefName: n.HeadRefName,
			BaseRefName: n.BaseRefName,
			Author:      n.Author.Login,
			Body:        n.Body,
			UpdatedAt:   n.UpdatedAt,
			Labels:      n.Labels.Nodes,
			Assignees:   assignees,
			Comments:    comments,
			HeadRepositoryOwner: models.GithubRepositoryOwner{
				Login: n.HeadRepositoryOwner.Login,
			},
		})
	}

	issues := make([]*models.GithubIssue, 0, len(parsed.Data.Repository.Issues.Nodes))
	for _, n := range parsed.Data.Repository.Issues.Nodes {
		assignees := make([]string, 0, len(n.Assignees.Nodes))
		for _, a := range n.Assignees.Nodes {
			assignees = append(assignees, a.Login)
		}
		comments := make([]models.GithubComment, 0, len(n.Comments.Nodes))
		for _, c := range n.Comments.Nodes {
			comments = append(comments, models.GithubComment{
				Author:    c.Author.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt,
			})
		}
		issues = append(issues, &models.GithubIssue{
			Number:    n.Number,
			Title:     n.Title,
			State:     n.State,
			Url:       n.Url,
			Author:    n.Author.Login,
			Body:      n.Body,
			UpdatedAt: n.UpdatedAt,
			Labels:    n.Labels.Nodes,
			Assignees: assignees,
			Comments:  comments,
		})
	}

	seenProjects := map[string]bool{}
	allProjects := make([]dashboardProjectNode, 0)
	for _, group := range [][]dashboardProjectNode{
		parsed.Data.Repository.ProjectsV2.Nodes,
		parsed.Data.RepositoryOwner.ProjectsV2.Nodes,
		parsed.Data.Viewer.ProjectsV2.Nodes,
	} {
		for _, p := range group {
			if p.ID == "" || seenProjects[p.ID] {
				continue
			}
			seenProjects[p.ID] = true
			allProjects = append(allProjects, p)
		}
	}

	var repoItems []*models.GithubProjectItem
	var otherItems []*models.GithubProjectItem

	for _, proj := range allProjects {
		var statusFieldID string
		var statusOptions []models.GithubProjectStatusOption
		for _, f := range proj.Fields.Nodes {
			if strings.EqualFold(f.Name, "Status") && f.ID != "" {
				statusFieldID = f.ID
				statusOptions = f.Options
				break
			}
		}

		for _, item := range proj.Items.Nodes {
			if item.ID == "" || item.Content.Title == "" {
				continue
			}
			var statusName, statusOptionID string
			if item.FieldValueByName != nil {
				statusName = item.FieldValueByName.Name
				statusOptionID = item.FieldValueByName.OptionID
			}
			url := item.Content.Url
			if url == "" {
				url = proj.Url
			}
			projItem := &models.GithubProjectItem{
				ItemID:            item.ID,
				ProjectID:         proj.ID,
				ProjectNumber:     proj.Number,
				ProjectTitle:      proj.Title,
				ProjectUrl:        proj.Url,
				StatusFieldID:     statusFieldID,
				StatusName:        statusName,
				StatusOptionID:    statusOptionID,
				StatusOptions:     statusOptions,
				ContentType:       item.Content.Typename,
				RepoNameWithOwner: item.Content.Repository.NameWithOwner,
				Number:            item.Content.Number,
				Title:             item.Content.Title,
				State:             item.Content.State,
				Url:               url,
				Author:            item.Content.Author.Login,
				Body:              item.Content.Body,
				Labels:            item.Content.Labels.Nodes,
			}
			if strings.EqualFold(projItem.RepoNameWithOwner, currentRepoFullName) {
				repoItems = append(repoItems, projItem)
			} else {
				otherItems = append(otherItems, projItem)
			}
		}
	}

	projectItems := append(repoItems, otherItems...)

	return &GithubDashboardData{
		PullRequests: prs,
		Issues:       issues,
		ProjectItems: projectItems,
	}, nil
}

func (self *GitHubCommands) UpdateProjectItemStatus(host string, token string, projectID string, itemID string, fieldID string, optionID string) error {
	endpoint := graphQLEndpoint(host)
	mutation := `
mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
  updateProjectV2ItemFieldValue(input: {
    projectId: $projectId
    itemId: $itemId
    fieldId: $fieldId
    value: { singleSelectOptionId: $optionId }
  }) {
    projectV2Item { id }
  }
}`
	variables := map[string]string{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   fieldID,
		"optionId":  optionID,
	}
	bodyBytes, err := json.Marshal(graphQLRequest{Query: mutation, Variables: variables})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyStr := new(bytes.Buffer)
		_, _ = bodyStr.ReadFrom(resp.Body)
		return fmt.Errorf("GraphQL mutation failed with status: %s. Body: %s", resp.Status, bodyStr.String())
	}
	return nil
}

func (self *GitHubCommands) CheckoutPRCmdObj(number int) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "pr", "checkout", strconv.Itoa(number)})
}

func (self *GitHubCommands) CommentPRCmdObj(number int) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "pr", "comment", strconv.Itoa(number), "--editor"})
}

func (self *GitHubCommands) ReviewPRCmdObj(number int, flag string) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "pr", "review", strconv.Itoa(number), flag})
}

func (self *GitHubCommands) MergePRCmdObj(number int, strategyFlag string) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "pr", "merge", strconv.Itoa(number), strategyFlag, "--delete-branch"})
}

func (self *GitHubCommands) ClosePRCmdObj(number int) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "pr", "close", strconv.Itoa(number)})
}

func (self *GitHubCommands) CommentIssueCmdObj(number int) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "issue", "comment", strconv.Itoa(number), "--editor"})
}

func (self *GitHubCommands) CreateIssueCmdObj() *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "issue", "create", "--editor"})
}

func (self *GitHubCommands) DevelopIssueCmdObj(number int) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "issue", "develop", strconv.Itoa(number), "--checkout"})
}

func (self *GitHubCommands) AssignIssueToMeCmdObj(number int) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "issue", "edit", strconv.Itoa(number), "--add-assignee", "@me"})
}

func (self *GitHubCommands) CloseIssueCmdObj(number int) *oscommands.CmdObj {
	return self.cmd.New([]string{"gh", "issue", "close", strconv.Itoa(number)})
}

func (self *GitHubCommands) EditItemLabelCmdObj(kind string, number int, add bool, label string) *oscommands.CmdObj {
	flag := "--add-label"
	if !add {
		flag = "--remove-label"
	}
	return self.cmd.New([]string{"gh", kind, "edit", strconv.Itoa(number), flag, label})
}

func (self *GitHubCommands) FetchRepoLabels() ([]models.GithubLabel, error) {
	output, _, err := self.cmd.New([]string{"gh", "label", "list", "--limit", "100", "--json", "name,color"}).DontLog().RunWithOutputs()
	if err != nil {
		return nil, err
	}
	var labels []models.GithubLabel
	if err := json.Unmarshal([]byte(output), &labels); err != nil {
		return nil, err
	}
	return labels, nil
}
