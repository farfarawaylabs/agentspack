## UI component best practices

- **Single Responsibility**: Each component should have one clear purpose and do it well.
- **Composability First**: Prefer building UIs by composing small components rather than adding more flags/props to one “mega component.”
- **Prefer Composition Over Prop Explosion**: If a component needs many props (especially booleans like `showX`, `enableY`, `isZ`), refactor to composition.
  - Split into smaller components, or provide subcomponents/slots (e.g., `Card.Header`, `Card.Body`, `Card.Footer`).
  - Prefer `children`/slots over `renderFoo`, `fooComponent`, or `fooClassName` props unless there’s a strong reason.
- **Avoid Too Many Boolean Props**: Multiple boolean props usually indicate multiple responsibilities. Replace with:
  - A single `variant`/`size`/`intent` prop (constrained set of options), and/or
  - Composition (optional child sections), and/or
  - Separate components for distinct behaviors.
- **Clear Interface**: Define explicit props with sensible defaults; keep prop names predictable and consistent.
- **Encapsulation**: Hide internal implementation details; expose only what consumers need (props/events/slots).
- **Consistent Naming**: Use descriptive names for components and props; follow team conventions.
- **State Management**: Keep state local by default; lift it only when multiple components must share it.
- **Controlled vs Uncontrolled**: Be consistent per component; support controlled usage when the state must be managed by parents.
- **Loading/Empty/Error States**: Components that depend on async data should define these states explicitly.
- **Accessibility**: Use semantic HTML; ensure keyboard navigation, focus management, and labels. Use ARIA only when needed.
- **Performance by Default**: Avoid unnecessary re-renders (stable keys; memoization when it matters; avoid creating new objects/functions in hot paths).
- **Documentation**: Document usage, props, examples, and edge cases (controlled/uncontrolled, async states, accessibility notes).

### Prop explosion warning signs (refactor triggers)

- More than ~8–10 props, or more than ~3 boolean flags.
- Props that only make sense in certain combinations (many conditional branches).
- Many style props (`headerClassName`, `bodyClassName`, `footerClassName`, etc.) to customize internal parts.
- “God components” that handle layout + data fetching + formatting + interactions.

### Preferred composable patterns

- **Compound components**: `Menu`, `Menu.Item`, `Menu.Separator`
- **Slot/children sections**: `Modal` with `ModalHeader`, `ModalBody`, `ModalFooter`
- **Presentational + container split**: `UserCard` (UI) + `UserCardContainer` (data/state)
- **Variant props for styling only**: `variant="primary" | "secondary"` instead of many flags
