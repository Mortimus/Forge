{{.SystemPrompt}}

# Resolution Task

## Context
Use the following context to understand the project's history and coding standards:
{{.AgentsMemory}}

## Objective
Execute the approved Implementation Plan.
Write the code to satisfy the Specs and the Plan.

## Specs
{{.SpecContent}}

## Approved Plan
{{.PlanContent}}

## Output
1. Implement all changes described in the plan.
2. Update the Implementation Plan at `{{.ImplementationPlanPath}}` to change `Status: Approved` to `Status: Completed`.
