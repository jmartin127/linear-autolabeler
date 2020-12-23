package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/jmartin127/linear-autolabeler/linear"
	"github.com/jmartin127/linear-autolabeler/sla"
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

type metricsSummary struct {
	totalTimeByState map[string]time.Duration
	countByState     map[string]int
}

func newMetricsSummary() metricsSummary {
	return metricsSummary{
		totalTimeByState: make(map[string]time.Duration, 0),
		countByState:     make(map[string]int, 0),
	}
}

func (ms *metricsSummary) printResults() {
	keys := make([]string, 0, len(ms.totalTimeByState))
	for k := range ms.totalTimeByState {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, state := range keys {
		totalTimeSpentHours := ms.totalTimeByState[state].Hours()
		numTickets := ms.countByState[state]
		avgTime := totalTimeSpentHours / float64(numTickets)

		fmt.Printf("%s\t%f\t%d\t%f\n", state, totalTimeSpentHours, numTickets, avgTime)
	}
}

func main() {
	fmt.Println("Starting metrics gathering...")

	if token == "" {
		log.Fatal("No auth token was provided.\nUsage: go run main.go -t <auth-token>")
	}
	lc := &linear.LinearClient{
		Token: token,
	}

	obTechLabelID, err := lc.FindLabelIDWithName(teamID, "OB Techs")
	if err != nil {
		log.Fatal(err)
	}

	var totalIssues int
	pagination := fmt.Sprintf("first:%d", pageSize)
	summary := newMetricsSummary()
	for true {
		fmt.Printf("Loading issues for team %s and page %s\n", teamID, pagination)
		response, err := lc.GetIssuesForTeam(teamID, pagination)
		if err != nil {
			log.Fatal(err)
		}

		for _, v := range response.Team.Issues.Edges {
			if v.IssueNode.State.Name == "Done" {

				hasObTechLabel, err := lc.TicketHasLabel(linear.TicketNumber(&v.IssueNode), obTechLabelID)
				if err != nil {
					log.Fatal(err)
				}
				if hasObTechLabel {
					issueMetrics := gatherMetricsFromIssue(&v.IssueNode)
					summary = addResultToSummary(summary, issueMetrics)
					totalIssues++
				}
			}
		}

		// pagination
		pagination = fmt.Sprintf(`first:%d after:"%s"`, pageSize, response.Team.Issues.PageInfo.EndCursor)
		if response.Team.Issues.PageInfo.HasNextPage == false {
			break
		}
	}

	fmt.Printf("Read %d issues\n", totalIssues)
	summary.printResults()
}

func addResultToSummary(summary metricsSummary, m map[string]time.Duration) metricsSummary {
	for k, v := range m {
		if _, ok := summary.totalTimeByState[k]; !ok {
			summary.totalTimeByState[k] = v
			summary.countByState[k] = 1
		} else {
			summary.totalTimeByState[k] = summary.totalTimeByState[k] + v
			summary.countByState[k] = summary.countByState[k] + 1
		}
	}

	return summary
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

		// remove weekends/holidays
		diff := sla.BusinessDurationBetweenTimes(stFirst.CreatedAt, stSecond.CreatedAt)

		if _, ok := results[attributeToState]; !ok {
			results[attributeToState] = diff
		} else {
			results[attributeToState] = results[attributeToState] + diff
		}
	}

	return results
}
