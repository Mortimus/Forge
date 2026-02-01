# Example Implementation Guide

This guide walks you through setting up Forge from scratch. It consists of two parts:
1.  **Server Setup**: Installing the Forge bot on a Linux server.
2.  **Repository Setup**: Preparing your GitHub project to be managed by Forge.

---

## Part 1: Server Setup 🖥️

We will install Forge as a `systemd` service.

### 1. Build & Install
First, build the binary and move it to a global location.

```bash
# Build
git clone https://github.com/mortimus/forge
cd forge
go build -o forge cmd/forge/main.go

# Install
sudo mkdir -p /opt/forge
sudo cp forge /opt/forge/
```

### 2. Create Service User
For security, create a restricted user for Forge.

```bash
sudo useradd -r -s /bin/false forge
sudo chown -R forge:forge /opt/forge
sudo chmod 750 /opt/forge
```

### 3. Configure Systemd Service
Copy the example service file and edit it.

```bash
sudo cp deployments/forge.service.example /etc/systemd/system/forge.service
sudo vim /etc/systemd/system/forge.service
```

### 4. Configuration
Forge now uses a YAML configuration file. Create `/opt/forge/config.yaml`:

```yaml
jules_api_key: "your_actual_google_key"
max_sessions_per_day: 100
check_interval_seconds: 60
state_file_path: "forge_state.json"
debug: false

repositories:
  - github_repo: "my-username/my-new-project"
    github_pat: "your_github_personal_access_token"
    auto_merge: true
    # Optional overrides
    # gap_analysis_template_path: "/opt/forge/templates/gap_analysis.md"
```

### 5. Start Forge
Enable the service so it starts on boot.

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now forge
sudo systemctl status forge
```

---

## Part 2: Repository Setup 📁

Now we set up the GitHub repository (`my-username/my-new-project`) that Forge is monitoring.

### 1. Initialize Structure
Clone your (empty) repo and create the standard directory structure.

```bash
git clone https://github.com/my-username/my-new-project
cd my-new-project

# Standard Forge folders
mkdir -p docs/spec
```

### 2. Implant the Brain 🧠
Forge needs to know *how* to code and *what* to remember.

**Create `SYSTEM_PROMPT.md`** (The "Personality"):
Content should define coding standards.
```markdown
# System Prompt
You are a Senior Rust Engineer.
- Prefer `thiserror` for library errors.
- Always write unit tests in the same file as logic.
- Use `cargo fmt`.
```

**Create `AGENTS.md`** (The "Context"):
Content starts empty but acts as a notepad.
```markdown
# Agents Context
- Project: Rust Calculator
- Current Goal: Implementation phase.
```

### 3. Create the First Spec 📝
Define what you want to build.

**Create `docs/spec/v1_core.md`**:
```markdown
# Spec: Core Logic
Implement a function `add(a: i32, b: i32) -> i32`.
It must return the sum.
```

### 4. The Trigger 🔫
Push these files to GitHub.

```bash
git add .
git commit -m "Initialize Forge project structure"
git push origin main
```

### 5. The Workflow Begins
1.  **Forge wakes up**: He sees `docs/spec/v1_core.md`.
2.  **Gap Analysis**: He creates a PR with `IMPLEMENTATION_PLAN.md`.
3.  **Review**: You merge the PR.
4.  **Approval**: You edit `IMPLEMENTATION_PLAN.md` on GitHub, changing `Status: Draft` to `Status: Approved`.
5.  **Resolution**: Forge sees the approval, writes the code, and opens a PR with the solution.
6.  **Done**: You merge the code.

---

## Part 3: Customizing Prompts (Optional) 🎨

If you want to drastically change how Forge thinks (beyond `SYSTEM_PROMPT.md`), you can override his internal prompt templates.

### 1. Get the Defaults
The default templates are available in the repository.

```bash
# Assuming you cloned the repo to /home/user/forge_source
cp /home/user/forge_source/examples/templates/*.md /opt/forge/templates/
```

### 2. Modify
Edit `/opt/forge/templates/gap_analysis.md` to change how Forge plans. For example, you could enforce that every plan must include a specific security checklist.

### 3. Configure Service
Update `/opt/forge/config.yaml` to tell Forge where to find the new files.

```yaml
repositories:
  - github_repo: "my-username/my-new-project"
    # ...
    gap_analysis_template_path: "templates/gap_analysis.md"
    resolution_template_path: "templates/resolution.md"
```

### 4. Restart
```bash
sudo systemctl restart forge
```

---

## Reference Material
Check out [examples/rust-calculator](examples/rust-calculator) for a complete reference project structure containing robust specs and configuration.
