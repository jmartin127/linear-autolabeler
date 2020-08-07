package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmartin127/linear-autolabeler/linear"
	"github.com/jmartin127/linear-autolabeler/sla"
)

func main() {
	// initialize the linear client
	if len(os.Args) < 2 {
		log.Fatal("No auth token was provided.\nUsage: go run main.go <auth-token>")
	}
	authToken := os.Args[1]
	lc := &linear.LinearClient{
		Token: authToken,
	}

	slaClient := sla.NewSLA(lc)

	// TODO load the Team ID, given the team name
	teamID := "99dea3d2-59ff-4273-b8a1-379d36bb1678"

	// find the "ExceedsSLA" label
	exceedsSLALabelID, err := lc.FindLabelIDWithName(teamID, "ExceedsSLA")
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
		response, err := lc.GetIssuesForTeam(teamID, pagination)
		if err != nil {
			log.Fatal(err)
		}

		for _, v := range response.Team.Issues.Edges {
			if v.IssueNode.State.Name == "Done" || v.IssueNode.State.Name == "Canceled" {
				continue
			}

			exceeds, durationExceeding, sla := slaClient.ExceedsSLA(&v.IssueNode, loc)
			ticketNumber := linear.TicketNumber(&v.IssueNode)
			if exceeds {
				addedLabel, err := lc.AddLabelToTicket(ticketNumber, exceedsSLALabelID)
				if err != nil {
					log.Fatal(err)
				}
				if addedLabel {
					comment := fmt.Sprintf("Uh oh!  This ticket is in the %s state, and exceeds the SLA by %s!  FYI, the SLA is %+v (in business hours).", v.IssueNode.State.Name, durationExceeding, sla)
					log.Printf("Ticket: %s, Adding Comment: %s\n", ticketNumber, comment)
					if err := lc.AddCommentToTicket(v.IssueNode.ID, comment); err != nil {
						log.Fatal(err)
					}
				}
			} else {
				if err := lc.RemoveLabelFromTicket(ticketNumber, exceedsSLALabelID); err != nil {
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
