First make sure you read the following documents to fully understand the project, the scope, the research, goals and proposed UI:

- .agents/product/PRD.md
- .agents/product/ProductPhases.md
- .agents/product/UIDesign.md

Then your goal is to write a detailed todo list that we can give to the development team to start and build the product.

** Guidelines **

- Break the list to distinct phases, each focused on specific feature or related set of features or screens.
- Break each phase to smaller distinct tasks. If possible, try to generate tasks that can be built in parallel so multiple developers can work on it at the same time.
- Assign a unique, stable ID or number to every task so it can be referenced from other documents and in dependency fields.
- Explicitly map task dependencies and the critical path. For each task, include "Prerequisites" and "Blocked by" fields so it’s clear what must happen first.
- Tag each task as Parallelizable or Sequential, and group parallelizable tasks into clearly named "Concurrency Groups" within each phase.
- Define clear interfaces/contracts early (API schemas, component props, data models) so frontend and backend can work in parallel; provide mocks/stubs where useful.
- At the end of each phase, add two lists: "Can run in parallel now" and "Must wait for" to make concurrency obvious at a glance.
- Make sure to separate tasks to frontend and backend.
- Make sure that by following the list we build the product in phases and in logical order, where each phase is built on top of the other.
- Make sure the todos start with a more modest MVP, and then adding the more advanced features on top of it.
- Prioritize MVP feature delivery first. Defer non-functional concerns (performance, logging/observability, security/hardening, scalability) to later phases unless they directly block the MVP.
- After every backend phase, make sure to have todos for writing tests for the backend API.

** Output **
Your output should be a ./agents/TODOS.md file in a format that allows to easily mark items that were done later.

- The ./agents/TODOS.md must:
  - Use checkboxes for tasks.
  - Include a stable Task ID/number for every task (e.g., T-001 or 1.2.3) and display it alongside the checkbox.
  - For every task, include: Lane (FE/BE/Design), Parallelizable (Yes/No), Depends on (task IDs) / Prerequisites, Blockers, Deliverable.
  - Clearly highlight parallelizable tasks (e.g., prefix with [P]) and group them under named Concurrency Groups per phase.
  - Include a brief dependency overview per phase (a simple ordered list or short diagram is enough).
  - Call out examples where sequencing matters (e.g., "User management API depends on database migrations") to avoid starting blocked work.

Additional instructions: $ARGUMENTS
