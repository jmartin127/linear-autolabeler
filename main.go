package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/machinebox/graphql"
	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

const (
	teamsQuery = `
	{
		teams {
		  nodes {
			id
			name
		  }
		}
	  }
	`

	issuesQuery = `{
		team(id: "%s") {
		  id
		  name
	  
		  issues(%s) {
			edges {
				node {
					id
					number
					createdAt
					title
					assignee {
						id
						name
					}
					state {
						id
						name
					}
					team {
						key
					}
					history {
						nodes {
							createdAt
							fromState {
								name
							}
							toState {
								name
							}
						}
					}
				}
				cursor
			}
			pageInfo {
				hasNextPage
				endCursor
			}
		  }
		}
	  }`

	issueCommentsQuery = `{
		issue(id: "%s") {
		  id
		  title
		  description
		  comments {
			nodes {
			  createdAt
			  body
			}
		  }
		}
	  }`

	workflowStatesQuery = `{
		workflowStates {
		  nodes{
			id
			name
		  }
		}
	  }`
)

type TeamIssuesResponse struct {
	Team Team `json:"team"`
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
}

type Assignee struct {
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

type IssueHistoryNode struct {
	CreatedAt time.Time     `json:"createdAt"`
	FromState WorkflowState `json:"fromState"`
	ToState   WorkflowState `json:"toState"`
}

type IssueCommentNode struct {
	CreatedAt time.Time `json:"createdAt"`
	Body      string    `json:"body"`
}

type WorkflowState struct {
	Name string `json:"name"`
}

type LinearClient struct {
	token string
}

func main() {
	// initialize the linear client
	if len(os.Args) < 2 {
		log.Fatal("No auth token was provided.\nUsage: go run main.go <auth-token>")
	}
	authToken := os.Args[1]
	lc := &LinearClient{
		token: authToken,
	}

	lc.getIssueComments("ISC-105")

	pagination := "first:50"
	var totalIssues int

	loc, err := time.LoadLocation("America/Denver")
	if err != nil {
		log.Fatal(err)
	}

	for true {
		query := fmt.Sprintf(issuesQuery, "99dea3d2-59ff-4273-b8a1-379d36bb1678", pagination)

		var response TeamIssuesResponse
		if err := lc.exectueQuery(query, &response); err != nil {
			log.Fatal(err)
		}

		for _, v := range response.Team.Issues.Edges {
			exceeds, durationExceeding := exceedsSLA(&v.IssueNode, loc)
			if exceeds {
				lastComment, err := lc.getLastTimeIssueWasCommentedOn(&v.IssueNode)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("Exceeds SLA: %+v, Ticket: %d, State: %s, TeamKey: %s, LastComment: %+v\n", durationExceeding, v.IssueNode.Number, v.IssueNode.State.Name, v.IssueNode.TeamName.Key, lastComment)
			}
			totalIssues++
		}

		pagination = fmt.Sprintf(`first:50 after:"%s"`, response.Team.Issues.PageInfo.EndCursor)
		query = fmt.Sprintf(issuesQuery, "99dea3d2-59ff-4273-b8a1-379d36bb1678", pagination)

		if response.Team.Issues.PageInfo.HasNextPage == false {
			break
		}
	}

	fmt.Printf("Total issues: %d\n", totalIssues)

	// var response interface{}
	// if err := exectueQuery(query, &response); err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("INterface: %+v\n", response)

}

func (lc *LinearClient) exectueQuery(query string, response interface{}) error {
	graphqlClient := graphql.NewClient("https://api.linear.app/graphql")
	graphqlRequest := graphql.NewRequest(query)

	headers := make(map[string][]string)
	headers["Authorization"] = []string{lc.token}
	graphqlRequest.Header = headers

	if err := graphqlClient.Run(context.Background(), graphqlRequest, &response); err != nil {
		return err
	}

	return nil
}

func exceedsSLA(issue *IssueNode, loc *time.Location) (bool, time.Duration) {

	if issue.State.Name == "Ready for Review" {
		return exceedsSLAInBusinessHours(issue, loc, "Ready for Review", 8)
	} else if issue.State.Name == "Accepted" {
		return exceedsSLAInBusinessHours(issue, loc, "Accepted", 16)
	} else if issue.State.Name == "In Progress" {
		return exceedsSLAInBusinessHours(issue, loc, "Accepted", 16)
	} else if issue.State.Name == "Additional Info Required" {
		return exceedsSLAInBusinessHours(issue, loc, "Additional Info Required", 16)
	}

	return false, time.Hour
}

func exceedsSLAInBusinessHours(issue *IssueNode, loc *time.Location, refState string, slaBusinessHours int) (bool, time.Duration) {
	timeEnteredCurrentState := getLastTimeIssueEnteredState(issue, refState)
	start := timeEnteredCurrentState.In(loc)
	end := time.Now().In(loc)
	durationInCurrentStateBusinessHours := businessDurationBetweenTimes(start, end)

	slaDuration := time.Hour * time.Duration(slaBusinessHours)
	if durationInCurrentStateBusinessHours > slaDuration {
		// determine how much the SLA is exceeded
		exceedsSLABySeconds := durationInCurrentStateBusinessHours.Seconds() - slaDuration.Seconds()
		return true, (time.Second * time.Duration(exceedsSLABySeconds))
	}

	return false, time.Hour
}

func (lc *LinearClient) getLastTimeIssueWasCommentedOn(issue *IssueNode) (time.Time, error) {
	ticketNumber := fmt.Sprintf("%s-%d", issue.TeamName.Key, issue.Number)
	comments, err := lc.getIssueComments(ticketNumber)
	if err != nil {
		return time.Time{}, err
	}

	lastCommentTime := time.Time{}
	for _, c := range comments {
		if lastCommentTime.IsZero() || c.CreatedAt.After(lastCommentTime) {
			lastCommentTime = c.CreatedAt
		}
	}

	return lastCommentTime, nil
}

func (lc *LinearClient) getIssueComments(ticketNumber string) ([]IssueCommentNode, error) {
	query := fmt.Sprintf(issueCommentsQuery, ticketNumber)

	var response IssueResponse
	if err := lc.exectueQuery(query, &response); err != nil {
		return nil, err
	}

	return response.Issue.IssueComments.Nodes, nil
}

func getTimeIssueEnteredCurrentState(issue *IssueNode) time.Time {
	return getLastTimeIssueEnteredState(issue, issue.State.Name)
}

func getLastTimeIssueEnteredState(issue *IssueNode, state string) time.Time {
	// It is possible that the issue was created within this state, and has never moved to another state
	timeEnteredState := issue.CreatedAt

	for _, history := range issue.IssueHistory.Nodes {
		if history.ToState.Name == state {
			if history.CreatedAt.After(timeEnteredState) {
				timeEnteredState = history.CreatedAt
			}
		}
	}

	return timeEnteredState
}

func businessDurationBetweenTimes(start, end time.Time) time.Duration {
	c := cal.NewBusinessCalendar()

	// add holidays that the business observes
	c.AddHoliday(
		us.NewYear,
		us.MemorialDay,
		us.IndependenceDay,
		us.LaborDay,
		us.ThanksgivingDay,
		us.ChristmasDay,
	)

	return c.WorkHoursInRange(start, end)
}
