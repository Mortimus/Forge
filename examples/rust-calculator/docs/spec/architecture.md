# Spec: System Architecture

This document outlines the high-level architecture and non-functional requirements for the `Rust Calculator`.

## 1. Directory Structure
The project must follow the standard Rust binary layout.

```text
.
├── Cargo.toml          # Dependencies and workspace config
├── README.md           # Instructions
├── src/
│   ├── main.rs         # Entry point, CLI parsing, wiring
│   ├── lib.rs          # (Optional) Core Logic module export
│   └── core/           # Core domain logic
│       ├── mod.rs      # internal module definition
│       ├── types.rs    # Operation enum, Calculation struct
│       └── ops.rs      # calculate() function
```

## 2. Component Diagram

```mermaid
graph TD
    User[User] -->|Invokes| CLI[CLI (src/main.rs)]
    CLI -->|Parses| Clap[Clap Arguments]
    CLI -->|Configures| Logger[Env Logger]
    CLI -->|Calls| Core[Core Logic (src/core)]
    Core -->|Returns| Result[f64 Result]
    CLI -->|Prints| Stdout[Standard Output]
    Logger -->|Logs| Stderr[Standard Error]
```

## 3. Design Principles

### 3.1. Separation of Concerns
- **CLI Layer**: Handles parsing, printing, and exit codes. Does NO math.
- **Core Layer**: Handles math. Pure functions. No `println!` allowed. Returns `Result`.

### 3.2. Performance
- **Startup Time**: Must be instant (< 50ms).
- **Binary Size**: Release build should be optimized (strip symbols if needed).

### 3.3. Dependencies
Keep the dependency tree small.
- `clap`
- `serde` + `serde_json` (for JSON output)
- `log` + `env_logger`
- `thiserror` (for libraries)
- `anyhow` (optional, for main)

## 4. Build & Release
- **Profile**: Use `release` profile for production builds.
- **LTO**: Enable Link Time Optimization in `Cargo.toml`.
