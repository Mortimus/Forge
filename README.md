# ![Forge Logo](logo.jpg) Forge - The Spec-Driven Development Bot 🤖

[![Build and Release](https://github.com/Mortimus/Forge/actions/workflows/release.yml/badge.svg)](https://github.com/Mortimus/Forge/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mortimus/Forge)](https://goreportcard.com/report/github.com/Mortimus/Forge)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/Mortimus/Forge)](https://github.com/Mortimus/Forge/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


Forge is a **stateless, autonomous coding agent** that lives in your GitHub repository. He transforms **Markdown Specifications** into **Working Code** through an infinite loop of analysis, planning, and execution.

Unlike a typical CI/CD tool, Forge doesn't just run tests—he writes the code that makes them pass.

## 🧠 Philosophy: Spec-Driven Development (SDD)

Forge enforces a strict workflow where **Documentation leads Implementation**. You never write code directly; you write the *specs* for the code you want.

1.  **Specs are Source of Truth**: `docs/spec/*.md` files define your application.
2.  **Plan before Action**: No code is written until an `IMPLEMENTATION_PLAN.md` exists.
3.  **Stateless**: Forge has no database. His memory is the repository itself (`AGENTS.md`, `IMPLEMENTATION_PLAN.md`).

## 🔄 The OODA Loop

Forge implements a continuous **Observe-Orient-Decide-Act** loop:

### 1. Observe (The Trigger)
Forge watches your repository for changes to:
- **Specs** (`docs/spec/*.md`): New requirements.
- **Plan** (`IMPLEMENTATION_PLAN.md`): Status updates or new drafts.

### 2. Orient (Gap Analysis)
When Specs change, Forge wakes up:
- **Analyzes** the difference between the Specs and the current Codebase.
- **Drafts** an `IMPLEMENTATION_PLAN.md` detailing exactly what files need to change.
- **Opens a PR** with this plan for review.
- **Auto-Approval**: If he finds a Draft plan in the `main` branch, he assumes it is approved and actionable.

### 3. Decide (Resolution)
Once an Implementation Plan is present (Draft or Approved):
- **Executes** the plan by writing/modifying code.
- **Opens a PR** with the finished code.

### 4. Act (Cleanup)
After the Resolution PR is merged:
- **Deletes** the `IMPLEMENTATION_PLAN.md` to indicate the cycle is complete.
- **Sleeps** until new Specs or Plans appear.

## 🚀 Getting Started

We provide a comprehensive guide to setting up Forge and your first project.

👉 **[Read the Example Implementation Guide](docs/EXAMPLE_IMPLEMENTATION.md)**

### Quick Links
- [Rust Calculator Example](examples/rust-calculator/): A full reference project structure.
- [Installation](docs/EXAMPLE_IMPLEMENTATION.md#part-1-server-setup): How to install the Forge service.

## ⚙️ Configuration

Forge is configured via a YAML configuration file (default: `config.yaml`).

| YAML Key | Description | Default |
|---|---|---|
| `jules_api_key` | Google Jules API Key | **Required** |
| `github_pat` | GitHub PAT with Repo scope (per repo) | **Required** |
| `github_repo` | Owner/Repo (e.g. `User/Repo`) | **Required** |
| `check_interval_seconds` | Polling interval when idle | `20` |
| `max_sessions_per_day` | Daily safety limit for Jules sessions | `100` |
| `state_file_path` | Path to persistent state file | `forge_state.json` |
| `debug` | Enable verbose logging | `false` |
| `auto_merge` | Automatically merge PRs | `true` |
| `spec_path` | Path to Spec files | `docs/spec` |
| `impl_plan_path` | Path to Implementation Plan | `IMPLEMENTATION_PLAN.md` |
| `agents_prompt_path` | Path to Context Memory | `AGENTS.md` |
| `system_prompt_path` | Path to System Prompt | `SYSTEM_PROMPT.md` |
| `gap_analysis_template_path` | Override Gap Analysis Prompt | *(optional)* |
| `resolution_template_path` | Override Resolution Prompt | *(optional)* |

## 🎨 Advanced: Custom Prompts

Forge uses **embedded prompt templates** for Gap Analysis and Resolution. You can override these to customize his behavior (e.g., to enforce specific architectural patterns or change the planning format).

1.  **Reference**: See [internal/prompts/templates](internal/prompts/templates) for the default templates.
2.  **Override**: Copy a template to your server, modify it, and set the `gap_analysis_template_path` or `resolution_template_path` in your `config.yaml` to point to it.

## 💾 State Persistence

Forge automatically persists its state to disk (default: `forge_state.json`). This ensures that:
- Active Jules sessions are tracked across restarts.
- Daily usage limits are preserved per repository.
- The application can be gracefully stopped and resumed without data loss.

## 🔍 Diagnostics

If you're having trouble connecting to Jules or resolving your repository, use the built-in diagnostic mode:

```bash
# List all sources recognized by your Jules API Key
./forge --list-sources
```

## 📄 License
MIT License. See [LICENSE](LICENSE.md).
