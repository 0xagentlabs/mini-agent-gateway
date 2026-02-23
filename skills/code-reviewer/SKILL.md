---
name: code-reviewer
description: Review code for quality, security, and best practices
user-invocable: true
---

## When to use

Use this skill when:
- User asks for code review
- User submits a PR or code snippet
- You detect potential issues in generated code

## Review checklist

Always check for:

### 1. Code Quality
- [ ] Clear variable/function names
- [ ] Proper error handling
- [ ] No dead code or unused imports
- [ ] Consistent formatting

### 2. Security
- [ ] No hardcoded secrets
- [ ] Input validation
- [ ] SQL injection prevention (if applicable)
- [ ] XSS prevention (if web-related)

### 3. Performance
- [ ] No unnecessary allocations
- [ ] Efficient algorithms
- [ ] Proper resource cleanup

### 4. Maintainability
- [ ] Comments for complex logic
- [ ] Test coverage
- [ ] Documentation

## How to review

1. **Read the code**: Use `fs:read` to load files
2. **Analyze**: Go through the checklist
3. **Provide feedback**: 
   - Start with positives
   - List issues with severity (🔴 High / 🟡 Medium / 🟢 Low)
   - Suggest specific improvements with code examples

## Example output

```
## Code Review: auth.go

### ✅ Positives
- Clean error handling
- Good use of context

### 🔴 High Priority
1. **Hardcoded JWT secret** (line 15)
   ```go
   // Bad
   secret := "my-secret-key"
   
   // Good
   secret := os.Getenv("JWT_SECRET")
   ```

### 🟡 Medium Priority
...
```
