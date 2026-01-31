{{.SystemPrompt}}

# Gap Analysis Task

## Context
Use the following context to understand the project's history and coding standards:
{{.AgentsMemory}}

## Objective
Analyze the provided Spec files against the current codebase state.
Identify ALL necessary changes to implement the requested features.

## Specs
{{.SpecContent}}

## Output
You must create or update the Implementation Plan at `{{.ImplementationPlanPath}}`.
The plan MUST start with `Status: Draft`.

Describe the plan in detail, listing every file that needs creation or modification.
Do NOT write code yet. Just plan.
