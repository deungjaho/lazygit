package presentation

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/samber/lo"
)

func GetPullRequestListDisplayStrings(prs []*models.GithubPullRequest) [][]string {
	return lo.Map(prs, func(pr *models.GithubPullRequest, _ int) []string {
		stateStyle := style.FgGreen
		switch pr.State {
		case "DRAFT":
			stateStyle = style.FgWhite
		case "MERGED":
			stateStyle = style.FgMagenta
		case "CLOSED":
			stateStyle = style.FgRed
		}

		checkBadge := ""
		switch pr.ChecksState {
		case "SUCCESS":
			checkBadge = style.FgGreen.Sprint("✓")
		case "FAILURE", "ERROR":
			checkBadge = style.FgRed.Sprint("✗")
		case "PENDING", "EXPECTED":
			checkBadge = style.FgYellow.Sprint("●")
		}

		numStr := stateStyle.Sprintf("#%d", pr.Number)
		if checkBadge != "" {
			numStr += " " + checkBadge
		}

		titleStr := theme.DefaultTextColor.Sprint(pr.Title)
		metaStr := style.FgCyan.Sprintf("@%s", pr.Author) + " " + style.FgYellow.Sprint(pr.HeadRefName)
		return []string{numStr, titleStr, metaStr}
	})
}

func GetIssueListDisplayStrings(issues []*models.GithubIssue) [][]string {
	return lo.Map(issues, func(issue *models.GithubIssue, _ int) []string {
		numStr := style.FgGreen.Sprintf("#%d", issue.Number)
		if issue.State == "CLOSED" {
			numStr = style.FgRed.Sprintf("#%d", issue.Number)
		}

		labelParts := lo.Map(issue.Labels, func(l models.GithubLabel, _ int) string {
			return "[" + l.Name + "]"
		})
		labelsStr := style.FgYellow.Sprint(strings.Join(labelParts, ""))

		titleStr := theme.DefaultTextColor.Sprint(issue.Title)
		if labelsStr != "" {
			titleStr += " " + labelsStr
		}
		authorStr := style.FgCyan.Sprintf("@%s", issue.Author)
		return []string{numStr, titleStr, authorStr}
	})
}

func GetProjectItemListDisplayStrings(items []*models.GithubProjectItem) [][]string {
	return lo.Map(items, func(item *models.GithubProjectItem, _ int) []string {
		statusName := item.StatusName
		if statusName == "" {
			statusName = "No Status"
		}
		statusStyle := style.FgYellow
		lower := strings.ToLower(statusName)
		if strings.Contains(lower, "done") || strings.Contains(lower, "complete") {
			statusStyle = style.FgGreen
		} else if strings.Contains(lower, "progress") || strings.Contains(lower, "doing") {
			statusStyle = style.FgCyan
		}

		projBadge := style.FgMagenta.Sprintf("[%s]", item.ProjectTitle)
		statusBadge := statusStyle.Sprintf("[%s]", statusName)

		idPrefix := ""
		if item.Number > 0 {
			idPrefix = style.FgGreen.Sprintf("#%d ", item.Number)
		}
		titleStr := idPrefix + theme.DefaultTextColor.Sprint(item.Title)
		return []string{projBadge, statusBadge, titleStr}
	})
}

func FormatPullRequestDetail(pr *models.GithubPullRequest) string {
	var b strings.Builder

	stateStyle := style.FgGreen.SetBold()
	switch pr.State {
	case "DRAFT":
		stateStyle = style.FgWhite.SetBold()
	case "MERGED":
		stateStyle = style.FgMagenta.SetBold()
	case "CLOSED":
		stateStyle = style.FgRed.SetBold()
	}

	b.WriteString(fmt.Sprintf("%s  %s\n",
		stateStyle.Sprintf("[%s #%d]", pr.State, pr.Number),
		style.AttrBold.Sprint(pr.Title),
	))
	b.WriteString(strings.Repeat("─", 64) + "\n")
	b.WriteString(fmt.Sprintf("%s %s   %s %s -> %s\n",
		style.FgWhite.Sprint("Author:"),
		style.FgCyan.Sprint("@"+pr.Author),
		style.FgWhite.Sprint("Branch:"),
		style.FgYellow.Sprint(pr.HeadRefName),
		style.FgGreen.Sprint(pr.BaseRefName),
	))
	if pr.ChecksState != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Checks:"), style.FgYellow.Sprint(pr.ChecksState)))
	}
	if len(pr.Labels) > 0 {
		names := lo.Map(pr.Labels, func(l models.GithubLabel, _ int) string { return "[" + l.Name + "]" })
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Labels:"), style.FgYellow.Sprint(strings.Join(names, " "))))
	}
	if len(pr.Assignees) > 0 {
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Assignees:"), style.FgCyan.Sprint(strings.Join(pr.Assignees, ", "))))
	}
	b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("URL:"), style.FgBlue.Sprint(pr.Url)))
	b.WriteString(strings.Repeat("─", 64) + "\n\n")

	if strings.TrimSpace(pr.Body) != "" {
		b.WriteString(pr.Body + "\n")
	} else {
		b.WriteString(style.FgWhite.Sprint("(No description provided)") + "\n")
	}

	if len(pr.Comments) > 0 {
		b.WriteString("\n" + strings.Repeat("═", 64) + "\n")
		b.WriteString(style.AttrBold.Sprintf("Comments (%d)\n", len(pr.Comments)))
		for _, c := range pr.Comments {
			b.WriteString("\n" + strings.Repeat("─", 48) + "\n")
			b.WriteString(fmt.Sprintf("%s  %s\n\n%s\n",
				style.FgCyan.SetBold().Sprint("@"+c.Author),
				style.FgWhite.Sprint(c.CreatedAt),
				c.Body,
			))
		}
	}

	return b.String()
}

func FormatIssueDetail(issue *models.GithubIssue) string {
	var b strings.Builder

	stateStyle := style.FgGreen.SetBold()
	if issue.State == "CLOSED" {
		stateStyle = style.FgRed.SetBold()
	}

	b.WriteString(fmt.Sprintf("%s  %s\n",
		stateStyle.Sprintf("[%s #%d]", issue.State, issue.Number),
		style.AttrBold.Sprint(issue.Title),
	))
	b.WriteString(strings.Repeat("─", 64) + "\n")
	b.WriteString(fmt.Sprintf("%s %s   %s %s\n",
		style.FgWhite.Sprint("Author:"),
		style.FgCyan.Sprint("@"+issue.Author),
		style.FgWhite.Sprint("Updated:"),
		style.FgWhite.Sprint(issue.UpdatedAt),
	))
	if len(issue.Labels) > 0 {
		names := lo.Map(issue.Labels, func(l models.GithubLabel, _ int) string { return "[" + l.Name + "]" })
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Labels:"), style.FgYellow.Sprint(strings.Join(names, " "))))
	}
	if len(issue.Assignees) > 0 {
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Assignees:"), style.FgCyan.Sprint(strings.Join(issue.Assignees, ", "))))
	}
	b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("URL:"), style.FgBlue.Sprint(issue.Url)))
	b.WriteString(strings.Repeat("─", 64) + "\n\n")

	if strings.TrimSpace(issue.Body) != "" {
		b.WriteString(issue.Body + "\n")
	} else {
		b.WriteString(style.FgWhite.Sprint("(No description provided)") + "\n")
	}

	if len(issue.Comments) > 0 {
		b.WriteString("\n" + strings.Repeat("═", 64) + "\n")
		b.WriteString(style.AttrBold.Sprintf("Comments (%d)\n", len(issue.Comments)))
		for _, c := range issue.Comments {
			b.WriteString("\n" + strings.Repeat("─", 48) + "\n")
			b.WriteString(fmt.Sprintf("%s  %s\n\n%s\n",
				style.FgCyan.SetBold().Sprint("@"+c.Author),
				style.FgWhite.Sprint(c.CreatedAt),
				c.Body,
			))
		}
	}

	return b.String()
}

func FormatProjectItemDetail(item *models.GithubProjectItem) string {
	var b strings.Builder

	statusName := item.StatusName
	if statusName == "" {
		statusName = "No Status"
	}

	b.WriteString(fmt.Sprintf("%s %s  %s\n",
		style.FgMagenta.SetBold().Sprintf("[Project #%d: %s]", item.ProjectNumber, item.ProjectTitle),
		style.FgCyan.SetBold().Sprintf("[%s]", statusName),
		style.AttrBold.Sprint(item.Title),
	))
	b.WriteString(strings.Repeat("─", 64) + "\n")
	if item.RepoNameWithOwner != "" {
		b.WriteString(fmt.Sprintf("%s %s", style.FgWhite.Sprint("Repo:"), style.FgYellow.Sprint(item.RepoNameWithOwner)))
		if item.Number > 0 {
			b.WriteString(fmt.Sprintf(" #%d (%s)", item.Number, item.ContentType))
		}
		b.WriteString("\n")
	}
	if item.Author != "" {
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Author:"), style.FgCyan.Sprint("@"+item.Author)))
	}
	if len(item.Labels) > 0 {
		names := lo.Map(item.Labels, func(l models.GithubLabel, _ int) string { return "[" + l.Name + "]" })
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Labels:"), style.FgYellow.Sprint(strings.Join(names, " "))))
	}
	if len(item.StatusOptions) > 0 {
		opts := lo.Map(item.StatusOptions, func(o models.GithubProjectStatusOption, _ int) string {
			if o.Name == item.StatusName {
				return style.FgGreen.SetBold().Sprint("● " + o.Name)
			}
			return "○ " + o.Name
		})
		b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("Columns:"), strings.Join(opts, "  ")))
	}
	b.WriteString(fmt.Sprintf("%s %s\n", style.FgWhite.Sprint("URL:"), style.FgBlue.Sprint(item.Url)))
	b.WriteString(strings.Repeat("─", 64) + "\n\n")

	if strings.TrimSpace(item.Body) != "" {
		b.WriteString(item.Body + "\n")
	} else {
		b.WriteString(style.FgWhite.Sprint("(No description provided)") + "\n")
	}

	return b.String()
}
