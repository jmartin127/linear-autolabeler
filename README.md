# Linear Auto-Labeler (Ticket Management Support)

## WIP

This project is currently a work in progress.  But it is now running within Docker, and executed on a schedule as a cron job on my local.  Will get this running in Kubernetes soon.

## Configuration Example

```yaml
team: "Integrations-Cases"
timeZone: "America/Denver"
ignoreIssueStates:
  - "Done"
  - "Canceled"
pageSize: 50
job:
  - name: "SLA: Taking too long to Verify the requested work was completed"
    filter:
      - type: SLA
        currentState: "Verify"
        longerThan: 8h
    action:
      label: "ExceedsSLA"
      comment: "Uh oh! This ticket is in the Verify state, and exceeds the SLA by ${slaExceeding}! FYI, the SLA is ${sla} (in business hours). Please verify that the work you requested has been completed, and close the ticket"
  - name: "SLA: Taking too long to complete tickets that are currently in progress"
    filter:
      - type: SLA
        currentState: "In Progress"
        enteredState: "Accepted"
        longerThan: 16h
    action:
      label: "ExceedsSLA"
      comment: "Eek! This ticket is in progress, but it exceeds the SLA by ${slaExceeding}! Let's get 'er caught up. FYI, the SLA is ${sla} (in business hours)."
  - name: "SLA: Taking too long to gather additional information needed in order to complete a ticket"
    filter:
      - type: SLA
        currentState: "Additional Info Required"
        longerThan: 16h
      - type: LastComment
        longerThan: 16h
    action:
      label: "ExceedsSLA"
      comment: "Dang gina! It is taking a long time to gather the additional information, exceeds the SLA by ${slaExceeding}! Please gather the required information, or comment on the ticket saying you have contacted the office. FYI, the SLA is ${sla} (in business hours)."

```

## References

* Linear API:
  * https://github.com/linearapp/linear/blob/master/docs/API.md
* Linear Schema:
  * https://github.com/linearapp/linear/blob/master/packages/sdk/src/schema.ts
* Insomnia Download:
  * https://insomnia.rest/download/#mac

## Reading Data from the Linear API

Listing out teams:
```bash
curl \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: <token>" \
  --data '{ "query": "{ teams { nodes { id name } } }" }' \
  https://api.linear.app/graphql
```

Example response:
```json
{
  "data": {
    "teams": {
      "nodes": [
        {
          "id": "99dea3d2-58ff-4273-b8a1-379d36bb1678",
          "name": "Team 1"
        },
        {
          "id": "2f3a837c-c892-4dca-a9e7-12031af86b2d",
          "name": "Team 1"
        }
      ]
    }
  }
}
```
