package linear

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
					user {
						id
						name
					}
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
