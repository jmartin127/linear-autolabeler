package linear

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/machinebox/graphql"
)

type LinearClient struct {
	Token string
}

func (lc *LinearClient) FindLabelIDWithName(teamID string, labelName string) (string, error) {
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

func (lc *LinearClient) GetIssuesForTeam(teamID string, pagination string) (*TeamIssuesResponse, error) {
	query := fmt.Sprintf(issuesQuery, teamID, pagination)

	var response TeamIssuesResponse
	if err := lc.exectueQuery(query, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (lc *LinearClient) AddLabelToTicket(ticketNumber string, labelID string) (bool, error) {
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

func (lc *LinearClient) AddCommentToTicket(ticketID string, comment string) error {
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

func (lc *LinearClient) RemoveLabelFromTicket(ticketNumber string, labelID string) error {
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

func TicketNumber(issue *IssueNode) string {
	return fmt.Sprintf("%s-%d", issue.TeamName.Key, issue.Number)
}

func (lc *LinearClient) GetLastTimeIssueWasCommentedOn(issue *IssueNode) (time.Time, error) {
	ticketNumber := TicketNumber(issue)
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

func GetLastTimeIssueEnteredState(issue *IssueNode, state string) time.Time {
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
	headers["Authorization"] = []string{lc.Token}

	graphqlRequest.Header = headers

	if err := graphqlClient.Run(context.Background(), graphqlRequest, &response); err != nil {
		return err
	}

	return nil
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
	return GetLastTimeIssueEnteredState(issue, issue.State.Name)
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
