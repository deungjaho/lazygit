package controllers

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type PullRequestsController struct {
	baseController
	*ListControllerTrait[*models.GithubPullRequest]
	c        *ControllerCommon
	showDiff bool
}

var _ types.IController = &PullRequestsController{}

func NewPullRequestsController(c *ControllerCommon) *PullRequestsController {
	return &PullRequestsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().PullRequests,
			c.Contexts().PullRequests.GetSelected,
			c.Contexts().PullRequests.GetSelectedItems,
		),
		c: c,
	}
}

func (self *PullRequestsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Select),
			Handler:           self.withItem(self.checkout),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Checkout PR branch (gh pr checkout)",
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('d'),
			Handler:           self.withItem(self.toggleDiff),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Toggle PR Diff / Overview in Main View",
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('c'),
			Handler:           self.withItem(self.comment),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Add comment in Editor",
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('V'),
			Handler:           self.withItem(self.reviewMenu),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Review PR (Approve / Request changes)",
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('M'),
			Handler:           self.withItem(self.mergeMenu),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Merge PR (Squash / Rebase / Merge)",
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('L'),
			Handler:           self.withItem(self.labelMenu),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Manage Labels",
			OpensMenu:         true,
		},
		{
			Keys:              menuKey('x'),
			Handler:           self.withItem(self.closePR),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Close PR",
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.OpenFile),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.OpenPullRequestInBrowser,
		},
	}
}

func (self *PullRequestsController) GetOnRenderToMain() func() {
	return func() {
		var task types.UpdateTask
		pr := self.context().GetSelected()
		title := "Pull Request"
		if pr == nil {
			task = types.NewRenderStringTask("No open pull requests")
		} else if self.showDiff {
			title = fmt.Sprintf("PR #%d Diff (press 'd' for Overview)", pr.Number)
			prefix := presentation.FormatPullRequestDetail(pr) + "\n" + style.FgYellow.Sprint("═══ Diff ═══") + "\n\n"
			cmd := exec.Command("gh", "pr", "diff", strconv.Itoa(pr.Number), "--color=always")
			task = types.NewRunCommandTaskWithPrefix(cmd, types.StaticPrefix(prefix))
		} else {
			title = fmt.Sprintf("PR #%d Overview (press 'd' for Diff)", pr.Number)
			task = types.NewRenderStringTask(presentation.FormatPullRequestDetail(pr))
		}

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: title,
				Task:  task,
			},
		})
	}
}

func (self *PullRequestsController) toggleDiff(_ *models.GithubPullRequest) error {
	self.showDiff = !self.showDiff
	self.GetOnRenderToMain()()
	return nil
}

func (self *PullRequestsController) checkout(pr *models.GithubPullRequest) error {
	self.c.WithWaitingStatus("Checking out PR...", func(gocui.Task) error {
		if err := self.c.Git().GitHub.CheckoutPRCmdObj(pr.Number).Run(); err != nil {
			return err
		}
		self.c.RefreshFromWorker(types.RefreshOptions{
			Scope: []types.RefreshableView{types.BRANCHES, types.COMMITS, types.FILES},
		})
		return nil
	})
	return nil
}

func (self *PullRequestsController) comment(pr *models.GithubPullRequest) error {
	if err := self.c.RunSubprocessAndRefresh(self.c.Git().GitHub.CommentPRCmdObj(pr.Number)); err != nil {
		return err
	}
	self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
	return nil
}

func (self *PullRequestsController) reviewMenu(pr *models.GithubPullRequest) error {
	items := []*types.MenuItem{
		{
			Label: "Approve PR (--approve)",
			OnPress: func() error {
				self.c.WithWaitingStatus("Approving PR...", func(gocui.Task) error {
					if err := self.c.Git().GitHub.ReviewPRCmdObj(pr.Number, "--approve").Run(); err != nil {
						return err
					}
					self.c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
					return nil
				})
				return nil
			},
		},
		{
			Label: "Comment Review in Editor (--comment)",
			OnPress: func() error {
				return self.c.RunSubprocessAndRefresh(self.c.OS().Cmd.New([]string{"gh", "pr", "review", strconv.Itoa(pr.Number), "--comment"}))
			},
		},
		{
			Label: "Request Changes in Editor (--request-changes)",
			OnPress: func() error {
				return self.c.RunSubprocessAndRefresh(self.c.OS().Cmd.New([]string{"gh", "pr", "review", strconv.Itoa(pr.Number), "--request-changes"}))
			},
		},
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title: fmt.Sprintf("Review PR #%d", pr.Number),
		Items: items,
	})
}

func (self *PullRequestsController) mergeMenu(pr *models.GithubPullRequest) error {
	runMerge := func(flag string) func() error {
		return func() error {
			self.c.WithWaitingStatus("Merging PR...", func(gocui.Task) error {
				if err := self.c.Git().GitHub.MergePRCmdObj(pr.Number, flag).Run(); err != nil {
					return err
				}
				self.c.RefreshFromWorker(types.RefreshOptions{
					Scope: []types.RefreshableView{types.PULL_REQUESTS, types.BRANCHES, types.COMMITS},
				})
				return nil
			})
			return nil
		}
	}
	items := []*types.MenuItem{
		{Label: "Squash and merge (--squash)", OnPress: runMerge("--squash")},
		{Label: "Rebase and merge (--rebase)", OnPress: runMerge("--rebase")},
		{Label: "Create merge commit (--merge)", OnPress: runMerge("--merge")},
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title: fmt.Sprintf("Merge PR #%d", pr.Number),
		Items: items,
	})
}

func (self *PullRequestsController) labelMenu(pr *models.GithubPullRequest) error {
	return showLabelMenu(self.c, "pr", pr.Number, pr.Labels)
}

func (self *PullRequestsController) closePR(pr *models.GithubPullRequest) error {
	self.c.Confirm(types.ConfirmOpts{
		Title:  "Close PR",
		Prompt: fmt.Sprintf("Close PR #%d (%s)?", pr.Number, pr.Title),
		HandleConfirm: func() error {
			self.c.WithWaitingStatus("Closing PR...", func(gocui.Task) error {
				if err := self.c.Git().GitHub.ClosePRCmdObj(pr.Number).Run(); err != nil {
					return err
				}
				self.c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
				return nil
			})
			return nil
		},
	})
	return nil
}

func (self *PullRequestsController) openInBrowser(pr *models.GithubPullRequest) error {
	return self.c.OS().OpenLink(pr.Url)
}

func (self *PullRequestsController) context() *context.PullRequestsContext {
	return self.c.Contexts().PullRequests
}

// --- IssuesController ---

type IssuesController struct {
	baseController
	*ListControllerTrait[*models.GithubIssue]
	c *ControllerCommon
}

var _ types.IController = &IssuesController{}

func NewIssuesController(c *ControllerCommon) *IssuesController {
	return &IssuesController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().Issues,
			c.Contexts().Issues.GetSelected,
			c.Contexts().Issues.GetSelectedItems,
		),
		c: c,
	}
}

func (self *IssuesController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:              menuKey('c'),
			Handler:           self.withItem(self.comment),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Add comment in Editor",
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('L'),
			Handler:           self.withItem(self.labelMenu),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Add / Remove Label",
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Select),
			Handler:           self.withItem(self.developBranch),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Create & checkout branch for Issue (gh issue develop)",
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('a'),
			Handler:           self.withItem(self.assignToMe),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Assign to @me",
			DisplayOnScreen:   true,
		},
		{
			Keys:            opts.GetKeys(opts.Config.Universal.New),
			Handler:         self.createIssue,
			Description:     "Create new Issue in Editor",
			DisplayOnScreen: true,
		},
		{
			Keys:              menuKey('x'),
			Handler:           self.withItem(self.closeIssue),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Close Issue",
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.OpenFile),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.OpenPullRequestInBrowser,
		},
	}
}

func (self *IssuesController) GetOnRenderToMain() func() {
	return func() {
		var task types.UpdateTask
		issue := self.context().GetSelected()
		title := "Issue"
		if issue == nil {
			task = types.NewRenderStringTask("No open issues")
		} else {
			title = fmt.Sprintf("Issue #%d", issue.Number)
			task = types.NewRenderStringTask(presentation.FormatIssueDetail(issue))
		}

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: title,
				Task:  task,
			},
		})
	}
}

func (self *IssuesController) comment(issue *models.GithubIssue) error {
	if err := self.c.RunSubprocessAndRefresh(self.c.Git().GitHub.CommentIssueCmdObj(issue.Number)); err != nil {
		return err
	}
	self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
	return nil
}

func (self *IssuesController) createIssue() error {
	if err := self.c.RunSubprocessAndRefresh(self.c.Git().GitHub.CreateIssueCmdObj()); err != nil {
		return err
	}
	self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
	return nil
}

func (self *IssuesController) labelMenu(issue *models.GithubIssue) error {
	return showLabelMenu(self.c, "issue", issue.Number, issue.Labels)
}

func (self *IssuesController) developBranch(issue *models.GithubIssue) error {
	self.c.WithWaitingStatus("Creating branch for issue...", func(gocui.Task) error {
		if err := self.c.Git().GitHub.DevelopIssueCmdObj(issue.Number).Run(); err != nil {
			return err
		}
		self.c.RefreshFromWorker(types.RefreshOptions{
			Scope: []types.RefreshableView{types.BRANCHES, types.COMMITS, types.FILES},
		})
		return nil
	})
	return nil
}

func (self *IssuesController) assignToMe(issue *models.GithubIssue) error {
	self.c.WithWaitingStatus("Assigning issue to @me...", func(gocui.Task) error {
		if err := self.c.Git().GitHub.AssignIssueToMeCmdObj(issue.Number).Run(); err != nil {
			return err
		}
		self.c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
		return nil
	})
	return nil
}

func (self *IssuesController) closeIssue(issue *models.GithubIssue) error {
	self.c.Confirm(types.ConfirmOpts{
		Title:  "Close Issue",
		Prompt: fmt.Sprintf("Close Issue #%d (%s)?", issue.Number, issue.Title),
		HandleConfirm: func() error {
			self.c.WithWaitingStatus("Closing issue...", func(gocui.Task) error {
				if err := self.c.Git().GitHub.CloseIssueCmdObj(issue.Number).Run(); err != nil {
					return err
				}
				self.c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
				return nil
			})
			return nil
		},
	})
	return nil
}

func (self *IssuesController) openInBrowser(issue *models.GithubIssue) error {
	return self.c.OS().OpenLink(issue.Url)
}

func (self *IssuesController) context() *context.IssuesContext {
	return self.c.Contexts().Issues
}

// --- ProjectsController ---

type ProjectsController struct {
	baseController
	*ListControllerTrait[*models.GithubProjectItem]
	c *ControllerCommon
}

var _ types.IController = &ProjectsController{}

func NewProjectsController(c *ControllerCommon) *ProjectsController {
	return &ProjectsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().Projects,
			c.Contexts().Projects.GetSelected,
			c.Contexts().Projects.GetSelectedItems,
		),
		c: c,
	}
}

func (self *ProjectsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Select),
			Handler:           self.withItem(self.changeStatusMenu),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Change Project Status (Todo / In Progress / Done)",
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              menuKey('s'),
			Handler:           self.withItem(self.changeStatusMenu),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       "Change Project Status",
			OpensMenu:         true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.OpenFile),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.OpenPullRequestInBrowser,
			DisplayOnScreen:   true,
		},
	}
}

func (self *ProjectsController) GetOnRenderToMain() func() {
	return func() {
		var task types.UpdateTask
		item := self.context().GetSelected()
		title := "Project Item"
		if item == nil {
			task = types.NewRenderStringTask("No GitHub Project items found")
		} else {
			title = fmt.Sprintf("Project: %s", item.ProjectTitle)
			task = types.NewRenderStringTask(presentation.FormatProjectItemDetail(item))
		}

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: title,
				Task:  task,
			},
		})
	}
}

func (self *ProjectsController) changeStatusMenu(item *models.GithubProjectItem) error {
	if item.StatusFieldID == "" || len(item.StatusOptions) == 0 {
		return errors.New("This project does not have a configurable Status field")
	}

	menuItems := lo.Map(item.StatusOptions, func(opt models.GithubProjectStatusOption, _ int) *types.MenuItem {
		selected := opt.ID == item.StatusOptionID || opt.Name == item.StatusName
		return &types.MenuItem{
			Label:  opt.Name,
			Widget: types.MakeMenuRadioButton(selected),
			OnPress: func() error {
				self.c.WithWaitingStatus("Updating project status...", func(gocui.Task) error {
					host := self.c.Model().GithubBaseHost
					if host == "" {
						host = "github.com"
					}
					token := self.c.Model().GithubAuthToken
					if token == "" {
						token = self.c.Git().GitHub.GetAuthToken(host)
					}
					if err := self.c.Git().GitHub.UpdateProjectItemStatus(
						host,
						token,
						item.ProjectID,
						item.ItemID,
						item.StatusFieldID,
						opt.ID,
					); err != nil {
						return err
					}
					item.StatusName = opt.Name
					item.StatusOptionID = opt.ID
					self.c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
					return nil
				})
				return nil
			},
		}
	})

	return self.c.Menu(types.CreateMenuOptions{
		Title: fmt.Sprintf("Set Status for: %s", item.Title),
		Items: menuItems,
	})
}

func (self *ProjectsController) openInBrowser(item *models.GithubProjectItem) error {
	return self.c.OS().OpenLink(item.Url)
}

func (self *ProjectsController) context() *context.ProjectsContext {
	return self.c.Contexts().Projects
}

func showLabelMenu(c *ControllerCommon, kind string, number int, currentLabels []models.GithubLabel) error {
	c.WithWaitingStatus("Fetching repository labels...", func(gocui.Task) error {
		repoLabels, err := c.Git().GitHub.FetchRepoLabels()
		if err != nil {
			return err
		}

		currentSet := map[string]bool{}
		for _, l := range currentLabels {
			currentSet[l.Name] = true
		}

		menuItems := lo.Map(repoLabels, func(l models.GithubLabel, _ int) *types.MenuItem {
			hasLabel := currentSet[l.Name]
			actionWord := "Add"
			if hasLabel {
				actionWord = "Remove"
			}
			return &types.MenuItem{
				LabelColumns: []string{l.Name, style.FgCyan.Sprint(actionWord)},
				Widget:       types.MakeMenuCheckBox(hasLabel),
				OnPress: func() error {
					c.WithWaitingStatus("Updating label...", func(gocui.Task) error {
						if err := c.Git().GitHub.EditItemLabelCmdObj(kind, number, !hasLabel, l.Name).Run(); err != nil {
							return err
						}
						c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.PULL_REQUESTS}})
						return nil
					})
					return nil
				},
			}
		})

		return c.Menu(types.CreateMenuOptions{
			Title: fmt.Sprintf("Toggle Labels on #%d", number),
			Items: menuItems,
		})
	})
	return nil
}
