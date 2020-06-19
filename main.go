package main

import (
	"context"
	"fmt"
	"log"

	"github.com/machinebox/graphql"
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
	  
		  issues {
			nodes {
			  id
			  title
			  assignee {
				id
				name
			  }
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
	Nodes []Node `json:"nodes"`
}

type Node struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Assignee Assignee `json:"assignee"`
}

type Assignee struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	query := fmt.Sprintf(issuesQuery, "99dea3d2-59ff-4273-b8a1-379d36bb1678")
	var teamIssuesResponse TeamIssuesResponse
	if err := exectueQuery(query, &teamIssuesResponse); err != nil {
		log.Fatal(err)
	}

	for _, i := range teamIssuesResponse.Team.Issues.Nodes {
		fmt.Printf("Issue %+v\n", i)
	}
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
