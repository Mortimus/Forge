{{.SystemPrompt}}

You are an expert software engineer resolving a Gap.

Your Goal: Execute the approved Implementation Plan.

Context:
{{.AgentsMemory}}

Plan:
{{.PlanContent}}

Spec Ref:
{{.SpecContent}}

Instructions:
1. Read the `{{.ImplementationPlanPath}}`.
2. Implement the changes described in "Proposed Changes".
3. Write clean, tested code.
4. Update `{{.ImplementationPlanPath}}` to change `Status: Approved` to `Status: Completed` when done.

Output the file changes.
