package models

import "fmt"

type GithubLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type GithubComment struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

type GithubPullRequest struct {
	HeadRefName         string                `json:"headRefName"`
	BaseRefName         string                `json:"baseRefName"`
	Number              int                   `json:"number"`
	Title               string                `json:"title"`
	State               string                `json:"state"` // "MERGED", "OPEN", "CLOSED", "DRAFT"
	ChecksState         string                `json:"checksState"`
	Url                 string                `json:"url"`
	Author              string                `json:"author"`
	Body                string                `json:"body"`
	UpdatedAt           string                `json:"updatedAt"`
	Labels              []GithubLabel         `json:"labels"`
	Assignees           []string              `json:"assignees"`
	Comments            []GithubComment       `json:"comments"`
	HeadRepositoryOwner GithubRepositoryOwner `json:"headRepositoryOwner"`
}

func (pr *GithubPullRequest) UserName() string {
	if pr.HeadRepositoryOwner.Login != "" {
		return pr.HeadRepositoryOwner.Login
	}
	return pr.Author
}

func (pr *GithubPullRequest) BranchName() string {
	return pr.HeadRefName
}

func (pr *GithubPullRequest) ID() string {
	return fmt.Sprintf("pr-%d", pr.Number)
}

func (pr *GithubPullRequest) URN() string {
	return pr.ID()
}

type GithubRepositoryOwner struct {
	Login string `json:"login"`
}

type GithubIssue struct {
	Number    int             `json:"number"`
	Title     string          `json:"title"`
	State     string          `json:"state"`
	Url       string          `json:"url"`
	Author    string          `json:"author"`
	Body      string          `json:"body"`
	UpdatedAt string          `json:"updatedAt"`
	Labels    []GithubLabel   `json:"labels"`
	Assignees []string        `json:"assignees"`
	Comments  []GithubComment `json:"comments"`
}

func (issue *GithubIssue) ID() string {
	return fmt.Sprintf("issue-%d", issue.Number)
}

func (issue *GithubIssue) URN() string {
	return issue.ID()
}

type GithubProjectStatusOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GithubProjectItem struct {
	ItemID            string                      `json:"itemId"`
	ProjectID         string                      `json:"projectId"`
	ProjectNumber     int                         `json:"projectNumber"`
	ProjectTitle      string                      `json:"projectTitle"`
	ProjectUrl        string                      `json:"projectUrl"`
	StatusFieldID     string                      `json:"statusFieldId"`
	StatusName        string                      `json:"statusName"`
	StatusOptionID    string                      `json:"statusOptionId"`
	StatusOptions     []GithubProjectStatusOption `json:"statusOptions"`
	ContentType       string                      `json:"contentType"` // "Issue", "PullRequest", "DraftIssue"
	RepoNameWithOwner string                      `json:"repoNameWithOwner"`
	Number            int                         `json:"number"`
	Title             string                      `json:"title"`
	State             string                      `json:"state"`
	Url               string                      `json:"url"`
	Author            string                      `json:"author"`
	Body              string                      `json:"body"`
	Labels            []GithubLabel               `json:"labels"`
}

func (item *GithubProjectItem) ID() string {
	return item.ItemID
}

func (item *GithubProjectItem) URN() string {
	return item.ItemID
}
