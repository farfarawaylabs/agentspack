# General Workflow

## Pre-Development Phase

If the .agentspack folder exists:

- Always make sure you read `.agentspack/prd.md` to understand the project scope. Make sure to also read the `.agentspack/TECHNICAL_REQUIREMENTS.md` to understand the technologies and requirements used in the project.
  You can also read `.agentspack/todos.md` to see what was done so far.

## Planning Phase (MANDATORY)

- **ALWAYS start by creating a detailed plan** before making any code changes
- **Decompose complex tasks** into smaller, manageable subtasks when possible
- Each subtask should be focused and specific (e.g., "Create user model", "Add authentication middleware", "Build login component")
- Mark the first task as "in_progress" and begin working

## Development Phase

- Always create a new branch before working on a new feature and commit changes when finished working
- Work on **one subtask at a time** from your plan
- After completing each coding subtask, **run a code review** focusing on the code that was just changed
- Never be lazy or take unexplained shortcuts. Think carefully, do complete work, and validate that each change actually solves the subtask.

## Code Review & Iteration Loop

- **After each coding task**, run a code review on the changes made
- If the reviewer suggests improvements:
  - **Implement the suggested changes immediately**
  - **Re-run the code review** on the updated code
  - **Continue this loop** until no important improvements are suggested
- Only move to the next subtask after the current one passes review

## Debugging & QA

- When investigating issues, do not jump to conclusions from the first symptom; gather evidence before selecting a fix.
- Rethink your current conclusion at least once and challenge whether alternative explanations better fit the evidence.
- Make sure you researched all relevant code paths, configs, logs, and assumptions before declaring a root cause.
- After applying a fix, verify with reproducible checks/tests that the original issue is resolved and no regression was introduced.
- If evidence conflicts with your current conclusion, stop, revise your hypothesis, and continue investigating.

## Post API Task

After finishing coding or updating any API endpoint:

- **Update the Postman collection** in the `postman/` folder at the project root. If the collection doesn't exist yet, create it. Ensure every endpoint includes full documentation: descriptions, all request parameters, headers, body schemas, and realistic example requests/responses for every field.
- **Update the `docs/API_GUIDE.md`** file with clear, easy-to-follow instructions on how to use the API. Include all endpoints, HTTP methods, URL parameters, query parameters, request bodies, response formats, and example usage. If the file doesn't exist yet, create it.

## Task Completion

- When finishing coding always run the build and check for any errors. If there are errors fix them before completing the task
- When finishing coding always check for type errors and fix any existing ones
- When finishing a task, make sure to mark it as completed in `.agentspack/todos.md` (add it if it's not there yet)
- When finishing a big section of the app (auth, db, api, etc) always add an .md file to the docs folder documenting what you did and how to use that code
- Before closing a task/session, reflect on whether you learned a reusable project-specific technique (debugging flow, error diagnosis pattern, project convention, comment style, etc.)
- If such reusable knowledge was learned, invoke the `create-cross-platform-skill` skill and create or update that skill in each relevant provider folder (`.cursor/skills/`, `.claude/skills/`, `.agents/skills/`) so it is available in future sessions
