# Example Implementation Guide

This guide walks you through setting up Ralph from scratch. It consists of two parts:
1.  **Server Setup**: Installing the Ralph bot on a Linux server.
2.  **Repository Setup**: Preparing your GitHub project to be managed by Ralph.

---

## Part 1: Server Setup 🖥️

We will install Ralph as a `systemd` service.

### 1. Build & Install
First, build the binary and move it to a global location.

```bash
# Build
git clone https://github.com/mortimus/ralph
cd ralph
go build -o ralph cmd/ralph/main.go

# Install
sudo mkdir -p /opt/ralph
sudo cp ralph /opt/ralph/
```

### 2. Create Service User
For security, create a restricted user for Ralph.

```bash
sudo useradd -r -s /bin/false ralph
sudo chown -R ralph:ralph /opt/ralph
sudo chmod 750 /opt/ralph
```

### 3. Configure Systemd Service
Copy the example service file and edit it.

```bash
sudo cp deployments/ralph.service /etc/systemd/system/
sudo vim /etc/systemd/system/ralph.service
```

**Crucial Configuration**:
You MUST set these variables in the `[Service]` section:

```ini
[Service]
# ...
Environment="JULES_API_KEY=your_actual_google_key"
Environment="GITHUB_PAT=your_github_personal_access_token"
Environment="GITHUB_REPO=my-username/my-new-project"
```

### 4. Start Ralph
Enable the service so it starts on boot.

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ralph
sudo systemctl status ralph
```

---

## Part 2: Repository Setup 📁

Now we set up the GitHub repository (`my-username/my-new-project`) that Ralph is monitoring.

### 1. Initialize Structure
Clone your (empty) repo and create the standard directory structure.

```bash
git clone https://github.com/my-username/my-new-project
cd my-new-project

# Standard Ralph folders
mkdir -p docs/spec
```

### 2. Implant the Brain 🧠
Ralph needs to know *how* to code and *what* to remember.

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
git commit -m "Initialize Ralph project structure"
git push origin main
```

### 5. The Workflow Begins
1.  **Ralph wakes up**: He sees `docs/spec/v1_core.md`.
2.  **Gap Analysis**: He creates a PR with `IMPLEMENTATION_PLAN.md`.
3.  **Review**: You merge the PR.
4.  **Approval**: You edit `IMPLEMENTATION_PLAN.md` on GitHub, changing `Status: Draft` to `Status: Approved`.
5.  **Resolution**: Ralph sees the approval, writes the code, and opens a PR with the solution.
6.  **Done**: You merge the code.

---

## Part 3: Customizing Prompts (Optional) 🎨

If you want to drastically change how Ralph thinks (beyond `SYSTEM_PROMPT.md`), you can override his internal prompt templates.

### 1. Get the Defaults
The default templates are available in the repository.

```bash
# Assuming you cloned the repo to /home/user/ralph_source
cp /home/user/ralph_source/examples/templates/*.md /opt/ralph/templates/
```

### 2. Modify
Edit `/opt/ralph/templates/gap_analysis.md` to change how Ralph plans. For example, you could enforce that every plan must include a specific security checklist.

### 3. Configure Service
Update `/etc/systemd/system/ralph.service` to tell Ralph where to find the new files.

```ini
[Service]
# ...
Environment="GAP_ANALYSIS_TEMPLATE_PATH=/opt/ralph/templates/gap_analysis.md"
Environment="RESOLUTION_TEMPLATE_PATH=/opt/ralph/templates/resolution.md"
```

### 4. Restart
```bash
sudo systemctl restart ralph
```

---

## Reference Material
Check out [examples/rust-calculator](examples/rust-calculator) for a complete reference project structure containing robust specs and configuration.
