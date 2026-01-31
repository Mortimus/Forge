{{.SystemPrompt}}

You are an expert software engineer performing a Gap Analysis.

Your Goal: Analyze the provided Spec against the current codebase and create an Implementation Plan.

Context:
{{.AgentsMemory}}

Spec:
{{.SpecContent}}

Instructions:
1. Analyze the Spec.
2. Check existing codebase context (if provided).
3. Create or Update `{{.ImplementationPlanPath}}` in the repository.
   - If the file exists, append/update it.
   - If new, create it.
4. The plan MUST include:
   - Goal Description
   - Proposed Changes (File by File)
   - Verification Plan
5. The plan MUST start with `Status: Draft`.

Output ONLY the file changes required to create/update the plan.
