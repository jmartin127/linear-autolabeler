package main

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
)

const (
	authToken  = "<auth-token-here>"
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
)

func main() {
	graphqlClient := graphql.NewClient("https://api.linear.app/graphql")
	graphqlRequest := graphql.NewRequest(teamsQuery)

	headers := make(map[string][]string)
	headers["Authorization"] = []string{authToken}
	graphqlRequest.Header = headers

	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		panic(err)
	}
	fmt.Println(graphqlResponse)
}
