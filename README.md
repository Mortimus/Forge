# ![Forge Logo](logo.jpg) Forge - The Spec-Driven Development Bot 🤖

[![Build and Release](https://github.com/Mortimus/Forge/actions/workflows/release.yml/badge.svg)](https://github.com/Mortimus/Forge/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mortimus/Forge)](https://goreportcard.com/report/github.com/Mortimus/Forge)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/Mortimus/Forge)](https://github.com/Mortimus/Forge/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Mortimus/Forge)](https://github.com/Mortimus/Forge)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/Mortimus/Forge/pulls)

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

### 3. Decide (Resolution)
Once an Implementation Plan is present:
- **Executes** the plan by writing/modifying code.
- **Automated Interaction**: Forge automatically approves internal plans and handles agent follow-up questions to keep the loop moving safely.
- **Opens a PR** with the finished code.

### 4. Act (Cleanup)
After the Resolution PR is merged:
- **Deletes** the `IMPLEMENTATION_PLAN.md` to indicate the cycle is complete.
- **Sleeps** until new Specs or Plans appear.

## 🛠️ Key Features

-   **Spec-Driven Automation**: Converts Markdown specs to PRs automatically.
-   **Intelligent Backpressure**: Built-in rate limiting and backpressure handlers for both GitHub and Google Jules APIs.
-   **Automated Feedback**: Automatically provides "Best Judgement" to Jules sessions when they await user input for minor decisions.
-   **Plan Approval Automation**: Automatically calls `:approvePlan` for tasks requiring plan confirmation.
-   **Stateless Persistence**: Lightweight state saving for session tracking across service restarts.

## 🚀 Getting Started

We provide a comprehensive guide to setting up Forge and your first project.

👉 **[Read the Example Implementation Guide](docs/EXAMPLE_IMPLEMENTATION.md)**

### Quick Links
- [Rust Calculator Example](examples/rust-calculator/): A full reference project structure.
- [Installation](docs/EXAMPLE_IMPLEMENTATION.md#part-1-server-setup): How to install the Forge service.

### Proxmox LXC Setup

To run Forge isolated in a Proxmox LXC container:

1. Create a new LXC container (Debian or Arch Linux template recommended).
2. Allocate minimal resources (e.g., 1 Core, 512MB RAM, 8GB Storage).
3. Start the container and open a console.
4. Download the latest Forge release and set up the systemd service:

```bash
# Update and install dependencies (Debian/Ubuntu example)
apt-get update && apt-get install -y wget curl systemd

# Download the latest Forge binary
wget https://github.com/Mortimus/Forge/releases/latest/download/forge-linux-amd64 -O /usr/local/bin/forge
chmod +x /usr/local/bin/forge

# Create configuration directory
mkdir -p /etc/forge

# Create your configuration file
cat << 'EOF' > /etc/forge/config.yaml
jules_api_key: "YOUR_JULES_API_KEY"
repositories:
  - github_repo: "Owner/Repo"
    github_pat: "YOUR_GITHUB_PAT"
EOF

# Set up the systemd service using our example template
# Customize paths if you are not running as root in /etc/forge
cat << 'EOF' > /etc/systemd/system/forge.service
[Unit]
Description=Forge SDD Bot
After=network.target

[Service]
ExecStart=/usr/local/bin/forge -config /etc/forge/config.yaml
WorkingDirectory=/etc/forge
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

# Enable and start the service
systemctl daemon-reload
systemctl enable --now forge.service
```

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

Forge uses **embedded prompt templates** for Gap Analysis and Resolution. You can override these to customize his behavior.

1.  **Reference**: See [internal/prompts/templates](internal/prompts/templates) for the default templates.
2.  **Override**: Copy a template to your server, modify it, and set the template paths in your `config.yaml`.

## 💾 State Persistence

Forge automatically persists its state to disk (default: `forge_state.json`). This ensures that active Jules sessions and usage limits are preserved across service restarts.

## 🔍 Diagnostics

Use the built-in diagnostic tools to verify connectivity:

```bash
# List all sources recognized by your Jules API Key
./forge --list-sources

# Delete all active Jules sessions (useful for cleanup)
./forge --delete-sessions
```

## 📄 License
MIT License. See [LICENSE](LICENSE.md).
