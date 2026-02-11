# Jules Sources API Reference

Detailed documentation for the Jules Sources API resource and its methods.

**Base URL**: `https://jules.googleapis.com`

---

## 1. Resource: Source

A source represents a repository or data set available to Jules.

### Fields
| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | `string` | Identifier. The full resource name. Format: `sources/{source_id}` |
| `id` | `string` | Output only. The short ID of the source. |
| `githubRepo` | `object (GitHubRepo)` | Union field. Data for a GitHub repository. |

---

## 2. Data Structures

### GitHubRepo
```json
{
  "owner": "string",
  "repo": "string",
  "isPrivate": "boolean",
  "defaultBranch": { "object (GitHubBranch)" },
  "branches": [ { "object (GitHubBranch)" } ]
}
```

### GitHubBranch
```json
{
  "displayName": "string"
}
```

---

## 3. Methods

### `list`
List available sources.
- **HTTP Request**: `GET /v1alpha/sources`
- **Output**: Returns a list of `Source` objects.

### `get`
Retrieve details of a specific source.
- **HTTP Request**: `GET /v1alpha/sources/{source_id}`
- **Parameters**: `source_id` (required).
