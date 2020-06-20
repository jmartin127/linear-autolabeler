package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/machinebox/graphql"
	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

const (
	authToken  = ""
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

type IssueNode struct {
	ID           string       `json:"id"`
	Number       int          `json:"number"`
	CreatedAt    time.Time    `json:"createdAt"`
	Title        string       `json:"title"`
	Assignee     Assignee     `json:"assignee"`
	State        State        `json:"state"`
	IssueHistory IssueHistory `json:"history"`
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

type IssueHistoryNode struct {
	CreatedAt time.Time     `json:"createdAt"`
	FromState WorkflowState `json:"fromState"`
	ToState   WorkflowState `json:"toState"`
}

type WorkflowState struct {
	Name string `json:"name"`
}

func main() {
	pagination := "first:50"
	var totalIssues int

	loc, err := time.LoadLocation("America/Denver")
	if err != nil {
		log.Fatal(err)
	}

	for true {
		query := fmt.Sprintf(issuesQuery, "99dea3d2-59ff-4273-b8a1-379d36bb1678", pagination)

		var response TeamIssuesResponse
		if err := exectueQuery(query, &response); err != nil {
			log.Fatal(err)
		}

		for _, v := range response.Team.Issues.Edges {
			exceeds, durationExceeding := exceedsSLA(&v.IssueNode, loc)
			if exceeds {
				fmt.Printf("Exceeds SLA: %+v, Ticket: %d, State: %s\n", durationExceeding, v.IssueNode.Number, v.IssueNode.State.Name)
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

func exectueQuery(query string, response interface{}) error {
	graphqlClient := graphql.NewClient("https://api.linear.app/graphql")
	graphqlRequest := graphql.NewRequest(query)

	headers := make(map[string][]string)
	headers["Authorization"] = []string{authToken}
	graphqlRequest.Header = headers

	if err := graphqlClient.Run(context.Background(), graphqlRequest, &response); err != nil {
		return err
	}

	return nil
}

func exceedsSLA(issue *IssueNode, loc *time.Location) (bool, time.Duration) {

	if issue.State.Name == "Ready for Review" {
		return exceedsSLAInBusinessHours(issue, loc, 8)
	} else if issue.State.Name == "In Progress" {
		return exceedsSLAInBusinessHours(issue, loc, 16)
	}

	return false, time.Hour
}

func exceedsSLAInBusinessHours(issue *IssueNode, loc *time.Location, slaBusinessHours int) (bool, time.Duration) {
	timeEnteredCurrentState := getTimeIssueEnteredCurrentState(issue)
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

func getTimeIssueEnteredCurrentState(issue *IssueNode) time.Time {
	// It is possible that the issue was created within this state, and has never moved to another state
	timeEnteredState := issue.CreatedAt

	for _, history := range issue.IssueHistory.Nodes {
		if history.ToState.Name == issue.State.Name {
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
