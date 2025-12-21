# Claude Code Specific Instructions

## Planning

Use the `TodoWrite` tool to create and track your task list. This is the recommended way to plan and track progress in Claude Code.

## Code Review

To run code reviews, use the `senior-code-reviewer` sub-agent. Invoke it after completing each coding subtask:

```
Use the senior-code-reviewer agent to review the changes I just made
```

The sub-agent will analyze the code for:
- Code quality and best practices
- Potential bugs or issues
- Performance considerations
- Security vulnerabilities
