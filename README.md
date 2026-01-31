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
2.  **Plan before Action**: No code is written until an `IMPLEMENTATION_PLAN.md` is approved by a human.
3.  **Stateless**: Forge has no database. His memory is the repository itself (`AGENTS.md`, `IMPLEMENTATION_PLAN.md`).

## 🔄 The OODA Loop

Forge implements a continuous **Observe-Orient-Decide-Act** loop:

### 1. Observe (The Trigger)
Forge watches your repository for changes to:
- **Specs** (`docs/spec/*.md`): New requirements.
- **Plan** (`IMPLEMENTATION_PLAN.md`): Human approval (`Status: Approved`).

### 2. Orient (Gap Analysis)
When Specs change, Forge wakes up:
- **Analyzes** the difference between the Specs and the current Codebase.
- **Consults** `AGENTS.md` for project context and "muscle memory".
- **Drafts** an `IMPLEMENTATION_PLAN.md` detailing exactly what files need to change.
- **Opens a PR** with this plan for your review.

### 3. Decide (Human Approval)
You review the PR.
- If the plan looks good, you merge it.
- You then edit `IMPLEMENTATION_PLAN.md` to change `Status: Draft` -> `Status: Approved`.
- You push this change.

### 4. Act (Resolution)
Forge sees the "Approved" status:
- **Executes** the plan by writing/modifying code.
- **Updates** `IMPLEMENTATION_PLAN.md` to `Status: Completed`.
- **Opens a PR** with the finished code.

## 🚀 Getting Started

We provide a comprehensive guide to setting up Forge and your first project.

👉 **[Read the Example Implementation Guide](docs/EXAMPLE_IMPLEMENTATION.md)**

### Quick Links
- [Rust Calculator Example](examples/rust-calculator/): A full reference project structure.
- [Installation](docs/EXAMPLE_IMPLEMENTATION.md#part-1-server-setup): How to install the Forge service.

## ⚙️ Configuration

Forge is configured via environment variables (typically in `/etc/systemd/system/forge.service`).

| Variable | Description | Default |
|---|---|---|
| `JULES_API_KEY` | Google Jules API Key | **Required** |
| `GITHUB_PAT` | GitHub PAT with Repo scope | **Required** |
| `GITHUB_REPO` | Owner/Repo (e.g. `User/Repo`) | **Required** |
| `AUTO_MERGE` | Automatically merge PRs | `true` |
| `SPEC_PATH` | Path to Spec files | `docs/spec` |
| `IMPL_PLAN_PATH` | Path to Implementation Plan | `IMPLEMENTATION_PLAN.md` |
| `AGENTS_PROMPT_PATH` | Path to Context Memory | `AGENTS.md` |
| `SYSTEM_PROMPT_PATH` | Path to System Prompt | `SYSTEM_PROMPT.md` |
| `GAP_ANALYSIS_TEMPLATE_PATH` | Override Gap Analysis Prompt | *(optional)* |
| `RESOLUTION_TEMPLATE_PATH` | Override Resolution Prompt | *(optional)* |

## 🎨 Advanced: Custom Prompts

Forge uses default prompt templates for Gap Analysis and Resolution. You can override these to customize his behavior (e.g., to enforce specific architectural patterns or change the planning format).

1.  **Reference**: See [examples/templates/](examples/templates/) for the default templates.
2.  **Override**: Copy a template to your server, modify it, and set the `GAP_ANALYSIS_TEMPLATE_PATH` or `RESOLUTION_TEMPLATE_PATH` environment variable in your service file to point to it.

## 📄 License
MIT License. See [LICENSE](LICENSE.md).
