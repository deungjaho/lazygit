package context

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type PullRequestsContext struct {
	*FilteredListViewModel[*models.GithubPullRequest]
	*ListContextTrait
}

var _ types.IListContext = (*PullRequestsContext)(nil)

func NewPullRequestsContext(c *ContextCommon) *PullRequestsContext {
	viewModel := NewFilteredListViewModel(
		func() []*models.GithubPullRequest { return c.Model().RepoPullRequests },
		func(pr *models.GithubPullRequest) []string {
			return []string{fmt.Sprintf("#%d", pr.Number), pr.Title, pr.Author, pr.HeadRefName}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetPullRequestListDisplayStrings(viewModel.GetItems())
	}

	return &PullRequestsContext{
		FilteredListViewModel: viewModel,
		ListContextTrait: &ListContextTrait{
			Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
				View:       c.Views().PullRequests,
				WindowName: "prs",
				Key:        PULL_REQUESTS_CONTEXT_KEY,
				Kind:       types.SIDE_CONTEXT,
				Focusable:  true,
			})),
			ListRenderer: ListRenderer{
				list:              viewModel,
				getDisplayStrings: getDisplayStrings,
			},
			c: c,
		},
	}
}

type IssuesContext struct {
	*FilteredListViewModel[*models.GithubIssue]
	*ListContextTrait
}

var _ types.IListContext = (*IssuesContext)(nil)

func NewIssuesContext(c *ContextCommon) *IssuesContext {
	viewModel := NewFilteredListViewModel(
		func() []*models.GithubIssue { return c.Model().Issues },
		func(issue *models.GithubIssue) []string {
			return []string{fmt.Sprintf("#%d", issue.Number), issue.Title, issue.Author}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetIssueListDisplayStrings(viewModel.GetItems())
	}

	return &IssuesContext{
		FilteredListViewModel: viewModel,
		ListContextTrait: &ListContextTrait{
			Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
				View:       c.Views().Issues,
				WindowName: "prs",
				Key:        ISSUES_CONTEXT_KEY,
				Kind:       types.SIDE_CONTEXT,
				Focusable:  true,
			})),
			ListRenderer: ListRenderer{
				list:              viewModel,
				getDisplayStrings: getDisplayStrings,
			},
			c: c,
		},
	}
}

type ProjectsContext struct {
	*FilteredListViewModel[*models.GithubProjectItem]
	*ListContextTrait
}

var _ types.IListContext = (*ProjectsContext)(nil)

func NewProjectsContext(c *ContextCommon) *ProjectsContext {
	viewModel := NewFilteredListViewModel(
		func() []*models.GithubProjectItem { return c.Model().ProjectItems },
		func(item *models.GithubProjectItem) []string {
			return []string{item.ProjectTitle, item.StatusName, fmt.Sprintf("#%d", item.Number), item.Title, item.RepoNameWithOwner}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetProjectItemListDisplayStrings(viewModel.GetItems())
	}

	return &ProjectsContext{
		FilteredListViewModel: viewModel,
		ListContextTrait: &ListContextTrait{
			Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
				View:       c.Views().Projects,
				WindowName: "prs",
				Key:        PROJECTS_CONTEXT_KEY,
				Kind:       types.SIDE_CONTEXT,
				Focusable:  true,
			})),
			ListRenderer: ListRenderer{
				list:              viewModel,
				getDisplayStrings: getDisplayStrings,
			},
			c: c,
		},
	}
}
