# General Workflow

## Pre-Development Phase

If the .agents folder exists:

- Always make sure you read `.agents/prd.md` to understand the project scope. Make sure to also read the `.agents/TECHNICAL_REQUIREMENTS.md` to understand the technologies and requirements used in the project.
  You can also read `.agents/todos.md` to see what was done so far.

## Planning Phase (MANDATORY)

- **ALWAYS start by creating a detailed plan** before making any code changes
- **Decompose complex tasks** into smaller, manageable subtasks when possible
- Each subtask should be focused and specific (e.g., "Create user model", "Add authentication middleware", "Build login component")
- Mark the first task as "in_progress" and begin working

## Development Phase

- Always create a new branch before working on a new feature and commit changes when finished working
- Work on **one subtask at a time** from your plan
- After completing each coding subtask, **run a code review** focusing on the code that was just changed

## Code Review & Iteration Loop

- **After each coding task**, run a code review on the changes made
- If the reviewer suggests improvements:
  - **Implement the suggested changes immediately**
  - **Re-run the code review** on the updated code
  - **Continue this loop** until no important improvements are suggested
- Only move to the next subtask after the current one passes review

## Task Completion

- When finishing coding always run the build and check for any errors. If there are errors fix them before completing the task
- When finishing coding always check for type errors and fix any existing ones
- When finishing a task, make sure to mark it as completed in `.agents/todos.md` (add it if it's not there yet)
- When finishing a big section of the app (auth, db, api, etc) always add an .md file to the docs folder documenting what you did and how to use that code
