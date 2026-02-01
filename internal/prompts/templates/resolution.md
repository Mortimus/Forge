{{.SystemPrompt}}

# Resolution Task

## Context
Use the following context to understand the project's history and coding standards:
{{.AgentsMemory}}

## Objective
Execute the approved Implementation Plan.
Write the code to satisfy the Specs and the Plan.

## Specs
Spec Files for Reference:
{{range .SpecFiles}}- {{.}}
{{end}}
Please read these files and use them as the primary source of truth for the implementation.

## Approved Plan
{{.PlanContent}}

## Output
1. Implement all changes described in the plan.
2. Ensure the code is complete and functional.
3. Do NOT update any "Status" lines in the implementation plan. The file will be automatically deleted after the PR is merged.
