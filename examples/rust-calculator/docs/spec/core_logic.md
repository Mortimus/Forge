# Spec: Core Logic Library (`rcalc_core`)

This document defines the requirements for the core mathematical logic of the calculator. It must be implemented as a separate library crate or module to ensure testability and reusability.

## 1. Domain Types

### 1.1. Enum `Operation`
Represents the supported mathematical operations.
- **Traits**: `Debug`, `Clone`, `Copy`, `PartialEq`, `Eq`
- **Variants**:
    - `Add` (`+`)
    - `Subtract` (`-`)
    - `Multiply` (`*`)
    - `Divide` (`/`)
- **Implementations**:
    - `std::str::FromStr`: Must parse string symbols (`+`, `-`, `*`, `/`) into variants. Return a descriptive error if parsing fails.
    - `std::fmt::Display`: Must format variants back to their string symbols.

### 1.2. Struct `Calculation`
Represents a single calculation request.
- **Traits**: `Debug`, `Clone`, `PartialEq`
- **Fields**:
    - `lhs`: `f64` (Left-hand side operand)
    - `rhs`: `f64` (Right-hand side operand)
    - `op`: `Operation` (The operation to perform)

## 2. API Definitions

### 2.1. Main Calculation Function
```rust
pub fn calculate(input: Calculation) -> Result<f64, MathError>
```
- **Behavior**:
    - `Add`: Returns `lhs + rhs`
    - `Subtract`: Returns `lhs - rhs`
    - `Multiply`: Returns `lhs * rhs`
    - `Divide`: Returns `lhs / rhs`
- **Constraints**:
    - Division by Zero: If `op` is `Divide` and `rhs` is `0.0` (or within epsilon of 0.0), return `MathError::DivisionByZero`.
    - Floating Point: Use standard IEEE 754 arithmetic.

## 3. Error Handling

### 3.1. Enum `MathError`
Must use the `thiserror` crate for ergonomic error handling.
- **Variants**:
    - `[error("division by zero is not allowed")] DivisionByZero`
    - `[error("unknown operation: {0}")] UnknownOperation(String)`
    - `[error("invalid number: {0}")] InvalidNumber(String)`

## 4. Testing Requirements

The implementation must include a comprehensive test suite in `src/lib.rs` (or `src/core.rs`).

### 4.1. Unit Tests
- **Basic Arithmetic**: Verify `1+1=2`, `10-5=5`, `3*4=12`, `10/2=5`.
- **Floating Point**: Verify precision handling (e.g., `0.1 + 0.2` approx `0.3`).
- **Negatives**: Verify handling of negative numbers (e.g., `-5 * -5 = 25`).
- **Zero**: `0 + 0`, `0 * 5`.

### 4.2. Edge Cases
- **Division by Zero**: Ensure `calculate` returns `Err(MathError::DivisionByZero)`.
- **Large Numbers**: Test with `f64::MAX` and `f64::MIN` to ensure no panics (overflows are normal in floats but should be consistent).
