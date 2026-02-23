---
name: commit
description: Generate a conventional commit message based on git diff
---

## /commit

When the user runs `/commit`, analyze the staged git changes and generate a conventional commit message.

### Steps

1. Run `git diff --cached` to see staged changes
2. Analyze the changes to determine:
   - Type: feat, fix, docs, style, refactor, test, chore
   - Scope: affected module/component (optional)
   - Description: concise summary (max 50 chars)
   - Body: detailed explanation if needed (wrap at 72 chars)
3. Output the formatted commit message

### Format

```
<type>(<scope>): <description>

<body>
```

### Examples

- `feat(auth): add OAuth2 login support`
- `fix(api): handle nil pointer in user handler`
- `docs(readme): update installation instructions`

Always use conventional commits format.
