# Email Agents

## Purpose

Use Email Routing, Email Sending, and the Agents SDK email hooks to build agents that can receive and send email as a native channel.

## Use email agents for

- personal assistant tasks
- support agents
- vendor/customer follow-ups
- asynchronous workflows
- task replies from third parties
- user-authenticated email flows

## Pattern

1. Email route receives inbound email.
2. Routing maps sender/thread/recipient to an Agent instance.
3. Agent `onEmail` parses message and attachments.
4. Agent correlates it to an existing task or starts a new one.
5. Agent updates state and SQL.
6. Agent replies through Email Sending when appropriate.
7. Agent schedules follow-up if no response arrives.

## Important design point

Email is not just a notification channel. For agent systems, inbound email is a wakeup event that can resume a waiting task.

## Source docs

- https://blog.cloudflare.com/email-for-agents/
- https://developers.cloudflare.com/agents/api-reference/email/
