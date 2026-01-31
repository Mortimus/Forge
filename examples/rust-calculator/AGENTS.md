# Agents Memory

This file serves as the "long-term memory" or context for the AI agents working on this project.

## Tech Stack & Preferences
- **Language**: Rust (Latest Stable)
- **CLI**: `clap` with `derive` feature.
- **Logging**: `log` facade + `env_logger`.
- **Testing**: Standard `cargo test`.
- **Formatting**: `cargo fmt` is mandatory.
- **Linting**: `cargo clippy` must pass.

## Project Structure
- Binary: `src/main.rs`
- Logic: `src/lib.rs` (preferred) or modules in `src/`.

## Workflow Notes
- Always check `docs/spec/` before implementing.
- Updates to `IMPLEMENTATION_PLAN.md` must be atomic and descriptive.
