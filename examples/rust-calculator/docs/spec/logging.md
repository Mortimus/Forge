# Spec: Logging & Observability

This document defines the logging strategy to ensure the application is debuggable and observable.

## 1. Technologies
- **Facade**: `log` crate.
- **Implementation**: `env_logger` crate.

## 2. Configuration
- **Default Level**: `WARN` (Quiet by default).
- **Verbose Level**: `DEBUG` (When `-v` or `--verbose` is passed).
- **Environment Override**: The application must respect the `RUST_LOG` environment variable if set, overriding the CLI flag defaults.

## 3. Log Taxonomy

### 3.1. Startup
- **Level**: `INFO`
- **Message**: "rcalc starting up version {version}"
- **When**: At the very beginning of `main`.

### 3.2. User Input
- **Level**: `DEBUG`
- **Message**: "Received command: {subcommand}, args: {args}"
- **When**: Immediately after argument parsing.

### 3.3. Calculation
- **Level**: `DEBUG`
- **Message**: "Performing calculation: {lhs} {op} {rhs}"
- **When**: Before calling the core logic.

### 3.4. Errors
- **Level**: `ERROR`
- **Message**: "Math error occurred: {error}"
- **When**: When a `MathError` (like division by zero) is caught in `main`.

## 4. Output Format
- Use the detailed default format of `env_logger` which includes timestamp and level.
- Logs must go to `stderr` so they do not pollute `stdout` (which is used for the result).

## 5. Implementation Details
- Initialize the logger *before* any other logic in `main`.
- Ensure `clap` logic sets the log level programmatically if `-v` is present using `env_logger::Builder`.
