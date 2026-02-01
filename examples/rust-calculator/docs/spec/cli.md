# Spec: CLI Interface (`rcalc`)

This document defines the Command Line Interface (CLI) for the calculator application. The CLI acts as the entry point for users and interacts with the `rcalc_core` library.

## 1. Technologies
- **Crate**: `clap` (latest version)
- **Features**: `derive` (for struct-based parsing)

## 2. CLI Structure

The application name is `rcalc`.

### 2.1. Arguments & Subcommands

| Command / Argument | Type | Required | Description | Example |
|---|---|---|---|---|
| `eval` | **Subcommand** | No | Evaluate a single quoted string expression. | `rcalc eval "1 + 2"` |
| `[expression]` | `String` | Yes (for eval) | The mathematical expression to parse. | `"10 / 2"` |
| `args` | **Subcommand** | No | Evaluate positional arguments. | `rcalc args 1 + 2` |
| `[lhs]` | `f64` | Yes (for args) | Left operand. | `10.5` |
| `[op]` | `char/String` | Yes (for args) | Operator (+, -, *, /). | `*` |
| `[rhs]` | `f64` | Yes (for args) | Right operand. | `2` |

### 2.2. Global Flags

| Flag | Short | Long | Description |
|---|---|---|---|
| Verbose | `-v` | `--verbose` | Sets logging level to DEBUG. |
| JSON Output | | `--json` | Outputs result in JSON format. |

## 3. Output Formats

### 3.1. Standard Output (Default)
Print ONLY the result number on a new line.
```text
2.5
```

### 3.2. JSON Output (`--json`)
Print a JSON object with the result and status.
```json
{
  "status": "success",
  "result": 2.5,
  "operation": "1 + 1.5"
}
```

### 3.3. Error Output
Print errors to `stderr`.
If `--json` is active, print a JSON error object to `stdout` (to keep parsing consistent).
```json
{
  "status": "error",
  "error": "division by zero"
}
```

## 4. Parser Logic

### 4.1. `eval` Subcommand
- Must split the input string by whitespace.
- Must support 3 distinct parts: `LHS`, `OP`, `RHS`.
- If parsing fails (e.g., "1+2" without spaces), return a helpful error message: "Please use spaces between arguments: '1 + 2'".

### 4.2. `args` Subcommand
- Parses arguments directly into types. `clap` should handle type validation (e.g., ensuring `lhs` is a float).

## 5. Help & Versioning
- `rcalc --help` must generate standard help text.
- `rcalc --version` must print the version from `Cargo.toml`.
