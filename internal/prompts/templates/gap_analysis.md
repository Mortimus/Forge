{{.SystemPrompt}}

# Gap Analysis Task

## Context
Use the following context to understand the project's history and coding standards:
{{.AgentsMemory}}

## Objective
Analyze the provided Spec files against the current codebase state.
Identify ALL necessary changes to implement the requested features.

## Specs
Spec Files to Analyze:
{{range .SpecFiles}}- {{.}}
{{end}}
Please read these files and use them as the primary source of truth for the implementation.

## Output
You must create or update the Implementation Plan at `{{.ImplementationPlanPath}}`.

1.  **Analyze** the gaps between the Specs and the Codebase.
2.  **Select** the SINGLE most impactful item to implement right now. Do not list every missing feature. Pick the one that provides the most value or unblocks the most work.
3.  **Plan** the implementation for that ONE item only.

The plan content should:
-   Briefly state the goal (the selected item).
-   List the files to create or modify.
-   Do NOT include any "Status" lines (e.g., "Status: Draft"). The plan is implicitly actionable once merged.
-   Do NOT write code yet. Just plan.
