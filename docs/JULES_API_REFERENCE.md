# Jules API Reference

**Base URL**: `https://jules.googleapis.com`
**Auth Header**: `X-Goog-Api-Key: YOUR_API_KEY`

## 1. Authentication
Obtain API Key from [Jules Settings](https://jules.google.com/settings#api).

**Service Endpoint**: `https://jules.googleapis.com`

---

## 2. Sources
See the detailed [Sources API Reference](JULES_SOURCES_API_REFERENCE.md) for full resource and method documentation.

### List Sources
Retrieve list of connected repositories.
`GET /v1alpha/sources`

**Response**:
```json
{
  "sources": [
    {
      "name": "sources/github/owner/repo",
      "id": "github/owner/repo",
      "githubRepo": { "owner": "owner", "repo": "repo" }
    }
  ]
}
```

## 3. Sessions (Tasks)
See the detailed [Sessions API Reference](JULES_SESSIONS_API_REFERENCE.md) for full resource and method documentation.

### Create Session (Trigger Task)
Start a new coding task.
`POST /v1alpha/sessions`

**Body**:
```json
{
  "title": "Task Title",
  "prompt": "Instruction for the agent",
  "sourceContext": {
    "source": "sources/github/owner/repo",
    "githubRepoContext": { "startingBranch": "main" }
  },
  "automationMode": "AUTO_CREATE_PR" 
}
```

- `automationMode`: `AUTO_CREATE_PR` triggers automated PR creation.

### List Sessions
`GET /v1alpha/sessions?pageSize=5`

### Get Session / Check Progress
`GET /v1alpha/sessions/{session_id}`

**Response** (includes outputs like PRs):
```json
{
  "name": "sessions/123...",
  "status": "...",
  "outputs": [
    {
      "pullRequest": {
        "url": "https://github.com/owner/repo/pull/1",
        "title": "PR Title"
      }
    }
  ]
}
```

### Approve Plan
If `requirePlanApproval` is set (default `false` via API?), you must approve manually.
`POST /v1alpha/sessions/{session_id}:approvePlan`

## 4. Integration Notes for Forge
- Forge uses `automationMode: AUTO_CREATE_PR`.
- Forge needs to find the correct valid `source` name (e.g., `sources/github/mortimus/Forge`) before creating a session, or construct it if it follows a standard pattern.
- Forge monitors the session via polling to see when `outputs.pullRequest` appears.

## 5. Changelog
- **v1alpha**: Initial release. Used by Forge v0.1+.
