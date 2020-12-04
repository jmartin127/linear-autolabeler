package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jmartin127/linear-autolabeler/linear"
)

var token string

func init() {
	flag.StringVar(&token, "t", "", "Linear Developer Token")
	flag.Parse()
}

const (
	pageSize = 50
	teamID   = "99dea3d2-59ff-4273-b8a1-379d36bb1678" // TODO load the team ID from the team name
)

func main() {
	fmt.Println("Starting metrics gathering...")

	if token == "" {
		log.Fatal("No auth token was provided.\nUsage: go run main.go -t <auth-token>")
	}
	lc := &linear.LinearClient{
		Token: token,
	}

	var totalIssues int
	pagination := fmt.Sprintf("first:%d", pageSize)
	summary := make(map[string]time.Duration, 0)
	for true {
		fmt.Printf("Loading issues for team %s and page %s\n", teamID, pagination)
		response, err := lc.GetIssuesForTeam(teamID, pagination)
		if err != nil {
			log.Fatal(err)
		}

		for _, v := range response.Team.Issues.Edges {
			ticketNumber := linear.TicketNumber(&v.IssueNode)
			if v.IssueNode.State.Name == "Done" {
				fmt.Printf("Ticket %s. State %s\n", ticketNumber, v.IssueNode.State.Name)
				issueMetrics := gatherMetricsFromIssue(&v.IssueNode)
				summary = combineDurations(summary, issueMetrics)
				totalIssues++
			}
		}

		// pagination
		pagination = fmt.Sprintf(`first:%d after:"%s"`, pageSize, response.Team.Issues.PageInfo.EndCursor)
		if response.Team.Issues.PageInfo.HasNextPage == false {
			break
		}
	}

	fmt.Printf("Read %d issues\n", totalIssues)
	printMap(summary)
}

func combineDurations(m1 map[string]time.Duration, m2 map[string]time.Duration) map[string]time.Duration {
	result := make(map[string]time.Duration, 0)

	for k, v := range m1 {
		if _, ok := result[k]; !ok {
			result[k] = v
		} else {
			result[k] = result[k] + v
		}
	}

	for k, v := range m2 {
		if _, ok := result[k]; !ok {
			result[k] = v
		} else {
			result[k] = result[k] + v
		}
	}

	return result
}

func printMap(m map[string]time.Duration) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		fmt.Printf("%s, %s\n", k, v.String())
	}
}

/*
Example:
2020-11-19 16:01:49.742 +0000 UTC, Ready for Review --> In Progress
2020-11-19 18:58:41.867 +0000 UTC, In Progress --> Additional Info Required
2020-11-19 20:39:22.856 +0000 UTC, Additional Info Required --> Waiting on Partner
2020-11-19 21:10:47.863 +0000 UTC, Waiting on Partner --> Ready for Review
2020-11-19 22:51:12.769 +0000 UTC, Ready for Review --> Accepted
2020-11-20 15:57:09.235 +0000 UTC, Accepted --> In Progress
2020-11-24 04:00:33.099 +0000 UTC, In Progress --> Verify
2020-11-24 19:23:14.224 +0000 UTC, Verify --> Done

Compare the time from the last to the one prior... and attribute the time to the "from" transition at the current index
*/
func gatherMetricsFromIssue(issue *linear.IssueNode) map[string]time.Duration {
	stateTransitions := make([]linear.IssueHistoryNode, 0)
	for _, history := range issue.IssueHistory.Nodes {
		if history.ToState.Name != "" {
			stateTransitions = append(stateTransitions, history)
		}
	}

	sort.Slice(stateTransitions, func(i, j int) bool {
		return stateTransitions[i].CreatedAt.Before(stateTransitions[j].CreatedAt)
	})

	// need at least two entries to compare timestamps
	if len(stateTransitions) <= 1 {
		return map[string]time.Duration{}
	}

	results := make(map[string]time.Duration, 0)
	for i := len(stateTransitions) - 1; i > 1; i-- {
		stFirst := stateTransitions[i-1]
		stSecond := stateTransitions[i]
		attributeToState := stSecond.FromState.Name

		// TODO need to remove weekends and holidays!
		diff := stSecond.CreatedAt.Sub(stFirst.CreatedAt)
		//fmt.Printf("Diff: %s To State: %s\n", diff.String(), attributeToState)

		if _, ok := results[attributeToState]; !ok {
			results[attributeToState] = diff
		} else {
			results[attributeToState] = results[attributeToState] + diff
		}
	}

	return results
}
