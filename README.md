# Linear Auto-Labeler (Ticket Management Support)

## WIP

This project is currently a work in progress.  But it is now being used as a cron job on my local.  Will get this running in Kubernetes soon.

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
