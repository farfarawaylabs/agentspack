# Workflow to follow

## 01. Preparation:

- If you didn't do it already, read .agents/prd.md, .agents/technical_requirements.md and .agents/todos.md to understand the project and current progress.
- Read any relevant files in the .agents/docs folder (if it exists) for prior implementation details.
- Ensure subagents read all relevant files in the .agents folder to grasp requirements and context.
- Create a new branch for the work, and when you finish the work, commit all changes and update .agents/todos.md when the task is finished. Ask the user if he wants to merge the branch to main.

## 02. Delegation:

- If delegating to a subagent, ensure they read the plan (see below) before making any code changes.

## 03. Planning gate (before coding):

1.  Check the ./agents/plans folder and see if a plan already exists for the task.
2.  If a plan does not exist, create it first. Name the file using the Task ID so it’s easy to find and connect to the todo task (e.g., ./agents/plans/T-001.md). The plan should be a detailed description of what the agent is about to do and how it plans to implement the task. After writing the plan, stop and allow the user to review it before proceeding.
3.  If a plan already exists, read it carefully and only then move to the implementation phase.

- Plan template (use for each sub-task as needed):

````markdown
# [Feature/Task Name] Implementation Plan

## Overview

[Brief description of what we're implementing and why]

## Current State Analysis

[What exists now, what's missing, key constraints discovered]

## Desired End State

[A Specification of the desired end state after this plan is complete, and how to verify it]

### Key Discoveries:

- [Important finding with file:line reference]
- [Pattern to follow]
- [Constraint to work within]

## What We're NOT Doing

[Explicitly list out-of-scope items to prevent scope creep]

## Implementation Approach

[High-level strategy and reasoning]

## Subtask 1: [Descriptive Name]

### Overview

[What this phase accomplishes]

### Changes Required:

#### 1. [Component/File Group]

**File**: `path/to/file.ext`
**Changes**: [Summary of changes]

```[language]
// Specific code to add/modify
```

## Subtask 2: [Descriptive Name]

### Overview

[What this phase accomplishes]

### Changes Required:

#### 1. [Component/File Group]

**File**: `path/to/file.ext`
**Changes**: [Summary of changes]

```[language]
// Specific code to add/modify
```
````

## 04. Code Review

Before finishing the work, conduct a comprehensive code review on all your changes. This is a critical quality gate that ensures code excellence.

### Code Review Process:

1. **Identify All Changes**

   - List all modified, added, or deleted files
   - Use git diff or similar tools to see the full scope of changes
   - Review each file systematically

2. **Code Quality Assessment**

   - **Readability**: Is the code clear and easy to understand? Are variable/function names descriptive?
   - **Maintainability**: Can others easily modify this code? Is it well-organized?
   - **Performance**: Are there any obvious performance issues or inefficiencies?
   - **Security**: Check for common vulnerabilities (SQL injection, XSS, exposed secrets, etc.)
   - **Error Handling**: Are errors properly caught and handled? Are edge cases covered?

3. **Best Practices Verification**

   - Does the code follow the project's coding standards and style guide?
   - Are there proper comments/documentation for complex logic?
   - Is the code DRY (Don't Repeat Yourself)? Look for duplications
   - Are functions/methods single-purpose and appropriately sized?
   - Is there proper separation of concerns?

4. **Integration & Dependencies**

   - Are all dependencies properly declared and versioned?
   - Does the code integrate properly with existing systems?
   - Are there any breaking changes that need documentation?
   - Are API contracts maintained (backward compatibility)?

5. **Documentation Check**

   - Is the code adequately commented where necessary?
   - Are README files or other docs updated to reflect changes?
   - Are API changes documented?
   - Are configuration changes documented?

6. **Linting & Static Analysis**

   - Run linters and fix any warnings/errors
   - Check for unused imports, variables, or functions
   - Verify consistent formatting throughout

7. **Create Review Summary**

   - Document findings in a structured format
   - Categorize issues by severity (critical, major, minor, suggestions)
   - List all fixes that need to be applied
   - Note any positive patterns worth highlighting

8. **Apply Fixes**

   - Address all critical and major issues immediately
   - Consider minor issues and apply if time permits
   - Evaluate suggestions and implement valuable ones
   - Re-review after making fixes to ensure no new issues introduced

9. **Delegation Option**
   - If you have a code-review subagent available, delegate this entire process to them
   - Ensure the subagent reviews against all points listed above
   - Review the subagent's findings and apply recommended changes

### Review Completion Checklist:

- [ ] All files have been reviewed
- [ ] Code meets quality standards
- [ ] All critical/major issues resolved
- [ ] Tests pass
- [ ] Linting passes
- [ ] Documentation is updated
- [ ] No obvious security concerns
- [ ] Code follows project conventions

Only proceed to commit and finish the work after completing this review process and addressing all findings.

## 05. Finish work

1. Commit all the changes to the branch.
2. Update the .agents/todos.md
3. Add a documents to the .agents/docs detailing your work, how to use it and how to test it.
4. Ask the user to review all changes and if he wants to merge the branch to main.

Addiotnal instructions: #$ARGUMENTS
