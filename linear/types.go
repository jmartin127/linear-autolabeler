package linear

import "time"

type TeamIssuesResponse struct {
	Team Team `json:"team"`
}

type TeamLabelsResponse struct {
	TeamLabels TeamLabels `json:"team"`
}

type TeamLabels struct {
	IssueLabels IssueLabels `json:"labels"`
}

type Team struct {
	Issues Issues `json:"issues"`
}

type TeamName struct {
	Key string `json:"key"`
}

type Issues struct {
	TotalCount int      `json:"totalCount"`
	Edges      []Edge   `json:"edges"`
	PageInfo   PageInfo `json:"pageInfo"`
}

type Edge struct {
	IssueNode IssueNode `json:"node"`
	Cursor    string    `json:"cursor"`
}

type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage`
	EndCursor   string `json:"endCursor"`
}

type IssueResponse struct {
	Issue IssueNode `json:"issue"`
}

type IssueNode struct {
	ID            string        `json:"id"`
	Number        int           `json:"number"`
	CreatedAt     time.Time     `json:"createdAt"`
	Title         string        `json:"title"`
	Assignee      Assignee      `json:"assignee"`
	State         State         `json:"state"`
	TeamName      TeamName      `json:"team"`
	IssueHistory  IssueHistory  `json:"history"`
	IssueComments IssueComments `json:"comments"`
	IssueLabels   IssueLabels   `json:"labels"`
}

type Assignee struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type State struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IssueHistory struct {
	Nodes []IssueHistoryNode `json:"nodes"`
}

type IssueComments struct {
	Nodes []IssueCommentNode `json:"nodes"`
}

type IssueLabels struct {
	Nodes []IssueLabelNode `json:"nodes"`
}

type IssueHistoryNode struct {
	CreatedAt time.Time     `json:"createdAt"`
	FromState WorkflowState `json:"fromState"`
	ToState   WorkflowState `json:"toState"`
}

type IssueCommentNode struct {
	CreatedAt time.Time `json:"createdAt"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
}

type IssueLabelNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkflowState struct {
	Name string `json:"name"`
}

type IssueUpdateResponse struct {
	IssueUpdate SuccessResponse `json:"issueUpdate"`
}

type CommentCreateResponse struct {
	CommentCreate SuccessResponse `json:"commentCreate"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}
