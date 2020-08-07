package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
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

	issueLabelsQuery = `{
		issue(id: "%s") {
			id
			labels {
				nodes {
					id
					name
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

	addIssueLabelMutation = `mutation {
		issueUpdate(
		  id: "%s",
		  input: {
			labelIds: [%s]
		  }
		) {
		  success
		  issue {
			id
			title
		  }
		}
	  }`

	addIssueCommentMutation = `mutation {
  commentCreate(
    input: {
      issueId: "%s"
      body: "%s"
    }
  ) {
    success
  }
}`

	labelsQuery = `{
		team(id: "%s") {
			id
			name
		
			labels {
				nodes {
					id
					name
				}
			}
		}
	}`
)

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

	// TODO load the Team ID, given the team name
	teamID := "99dea3d2-59ff-4273-b8a1-379d36bb1678"

	// find the "ExceedsSLA" label
	exceedsSLALabelID, err := lc.findLabelIDWithName(teamID, "ExceedsSLA")
	if err != nil {
		log.Fatal(err)
	}

	pagination := "first:50"
	var totalIssues int

	// TODO the timezone should be configurable
	loc, err := time.LoadLocation("America/Denver")
	if err != nil {
		log.Fatal(err)
	}

	for true {
		query := fmt.Sprintf(issuesQuery, teamID, pagination)

		var response TeamIssuesResponse
		if err := lc.exectueQuery(query, &response); err != nil {
			log.Fatal(err)
		}

		for _, v := range response.Team.Issues.Edges {
			if v.IssueNode.State.Name == "Done" || v.IssueNode.State.Name == "Canceled" {
				continue
			}

			exceeds, durationExceeding, sla := lc.exceedsSLA(&v.IssueNode, loc)
			ticketNumber := ticketNumber(&v.IssueNode)
			if exceeds {
				addedLabel, err := lc.addLabelToTicket(ticketNumber, exceedsSLALabelID)
				if err != nil {
					log.Fatal(err)
				}
				if addedLabel {
					comment := fmt.Sprintf("Uh oh! This ticket is in the %s state, and exceeds SLA by %s! FYI, the SLA is %+v (in business hours).", v.IssueNode.State.Name, durationExceeding, sla)
					log.Printf("Ticket: %s, Adding Comment: %s\n", ticketNumber, comment)
					if err := lc.addCommentToTicket(v.IssueNode.ID, comment); err != nil {
						log.Fatal(err)
					}
				}
			} else {
				if err := lc.removeLabelFromTicket(ticketNumber, exceedsSLALabelID); err != nil {
					log.Fatal(err)
				}
			}
			totalIssues++
		}

		// pagination
		pagination = fmt.Sprintf(`first:50 after:"%s"`, response.Team.Issues.PageInfo.EndCursor)
		if response.Team.Issues.PageInfo.HasNextPage == false {
			break
		}
	}

	fmt.Printf("Total issues: %d\n", totalIssues)
}

func (lc *LinearClient) findLabelIDWithName(teamID string, labelName string) (string, error) {
	labels, err := lc.getTeamLabels(teamID)
	if err != nil {
		return "", err
	}

	for _, l := range labels {
		if l.Name == labelName {
			return l.ID, nil
		}
	}

	return "", fmt.Errorf("cannot find label with name %s", labelName)
}

func (lc *LinearClient) getTeamLabels(teamID string) ([]IssueLabelNode, error) {
	query := fmt.Sprintf(labelsQuery, teamID)

	var response TeamLabelsResponse
	err := lc.exectueQuery(query, &response)
	if err != nil {
		return nil, err
	}

	return response.TeamLabels.IssueLabels.Nodes, nil
}

func (lc *LinearClient) addLabelToTicket(ticketNumber string, labelID string) (bool, error) {
	// get current set of labels
	labels, err := lc.getLabels(ticketNumber)
	if err != nil {
		return false, err
	}

	// get the labelIDs from the labels
	labelIDs := make([]string, 0)
	for _, l := range labels {

		// if this label already exists, do not add it again
		if l.ID == labelID {
			return false, nil
		}
		labelIDs = append(labelIDs, l.ID)
	}

	// add the new label to the list
	labelIDs = append(labelIDs, labelID)

	// apply the labels
	fmt.Printf("Adding label to ticket: %s\n", ticketNumber)
	if err := lc.applyLabels(ticketNumber, labelIDs); err != nil {
		return false, err
	}

	return true, nil
}

func (lc *LinearClient) addCommentToTicket(ticketID string, comment string) error {
	mutation := fmt.Sprintf(addIssueCommentMutation, ticketID, comment)

	var response CommentCreateResponse
	err := lc.exectueQuery(mutation, &response)
	if err != nil {
		return err
	}

	if !response.CommentCreate.Success {
		return fmt.Errorf("Adding comment did not succeed for ticket with ID %s", ticketID)
	}

	return nil
}

func (lc *LinearClient) removeLabelFromTicket(ticketNumber string, labelID string) error {
	// get current set of labels
	labels, err := lc.getLabels(ticketNumber)
	if err != nil {
		return err
	}

	// get the labelIDs from the labels
	labelIDs := make([]string, 0)
	var foundLabel bool
	for _, l := range labels {
		// do not add the label that we are removing to the list
		if l.ID == labelID {
			foundLabel = true
		} else {
			labelIDs = append(labelIDs, l.ID)
		}
	}

	if foundLabel { // no need to apply the labels if it wasn't in the list
		// apply the labels
		fmt.Printf("Found label, removing from ticket %s\n", ticketNumber)
		return lc.applyLabels(ticketNumber, labelIDs)
	}

	return nil
}

func (lc *LinearClient) applyLabels(ticketNumber string, labelIDs []string) error {
	var labelIDsString string
	if len(labelIDs) > 0 {
		labelIDsString = strings.Join(labelIDs, `", "`)
		labelIDsString = fmt.Sprintf(`"%s"`, labelIDsString) // need this format: "id1", "id2", "id3"
	}
	mutation := fmt.Sprintf(addIssueLabelMutation, ticketNumber, labelIDsString)

	var response IssueUpdateResponse
	err := lc.exectueQuery(mutation, &response)
	if err != nil {
		return err
	}

	if !response.IssueUpdate.Success {
		return fmt.Errorf("Applying labels did not succeed for ticket %s", ticketNumber)
	}

	return nil
}

func (lc *LinearClient) getLabels(ticketNumber string) ([]IssueLabelNode, error) {
	query := fmt.Sprintf(issueLabelsQuery, ticketNumber)

	var response IssueResponse
	err := lc.exectueQuery(query, &response)
	if err != nil {
		return nil, err
	}

	return response.Issue.IssueLabels.Nodes, nil
}

func (lc *LinearClient) exectueQuery(query string, response interface{}) error {
	graphqlClient := graphql.NewClient("https://api.linear.app/graphql") // TODO only do this once in the client itself
	graphqlRequest := graphql.NewRequest(query)

	headers := make(map[string][]string)
	headers["Authorization"] = []string{lc.token}

	graphqlRequest.Header = headers

	if err := graphqlClient.Run(context.Background(), graphqlRequest, &response); err != nil {
		return err
	}

	return nil
}

func (lc *LinearClient) exceedsSLA(issue *IssueNode, loc *time.Location) (bool, time.Duration, time.Duration) {
	if issue.State.Name == "Ready for Review" {
		return exceedsSLAInBusinessHours(issue, loc, "Ready for Review", time.Hour*time.Duration(8))
	} else if issue.State.Name == "Accepted" {
		return exceedsSLAInBusinessHours(issue, loc, "Accepted", time.Hour*time.Duration(16))
	} else if issue.State.Name == "In Progress" {
		return exceedsSLAInBusinessHours(issue, loc, "Accepted", time.Hour*time.Duration(16))
	} else if issue.State.Name == "Verify" {
		return exceedsSLAInBusinessHours(issue, loc, "Verify", time.Hour*time.Duration(8))
	} else if issue.State.Name == "Waiting on Partner" {
		return exceedsSLAInBusinessHours(issue, loc, "Waiting on Partner", time.Hour*time.Duration(80))
	} else if issue.State.Name == "Additional Info Required" {
		exceedsSLA, _, _ := exceedsSLAInBusinessHours(issue, loc, "Additional Info Required", time.Hour*time.Duration(16))
		if exceedsSLA {
			lastCommentTime, err := lc.getLastTimeIssueWasCommentedOn(issue)
			if err != nil {
				log.Fatal(err) // TODO
			}
			exceedsSLAForCommentToBeAdded, timeOverdueForComment, sla := exceedsSLAInBusinessHoursForStart(lastCommentTime, loc, time.Hour*time.Duration(16))
			if exceedsSLAForCommentToBeAdded {
				return exceedsSLAForCommentToBeAdded, timeOverdueForComment, sla
			}
			return false, time.Hour, time.Hour
		}
		return false, time.Hour, time.Hour
	}

	return false, time.Hour, time.Hour
}

func exceedsSLAInBusinessHours(issue *IssueNode, loc *time.Location, refState string, sla time.Duration) (bool, time.Duration, time.Duration) {
	timeEnteredCurrentState := getLastTimeIssueEnteredState(issue, refState)
	return exceedsSLAInBusinessHoursForStart(timeEnteredCurrentState, loc, sla)
}

func exceedsSLAInBusinessHoursForStart(refTime time.Time, loc *time.Location, sla time.Duration) (bool, time.Duration, time.Duration) {
	start := refTime.In(loc)
	end := time.Now().In(loc)
	durationInCurrentStateBusinessHours := businessDurationBetweenTimes(start, end)

	if durationInCurrentStateBusinessHours > sla {
		// determine how much the SLA is exceeded
		exceedsSLABySeconds := durationInCurrentStateBusinessHours.Seconds() - sla.Seconds()
		return true, (time.Second * time.Duration(exceedsSLABySeconds)), sla
	}

	return false, time.Hour, time.Hour
}

func ticketNumber(issue *IssueNode) string {
	return fmt.Sprintf("%s-%d", issue.TeamName.Key, issue.Number)
}

func (lc *LinearClient) getLastTimeIssueWasCommentedOn(issue *IssueNode) (time.Time, error) {
	ticketNumber := ticketNumber(issue)
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
