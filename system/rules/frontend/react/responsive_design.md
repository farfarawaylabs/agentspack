## Responsive design best practices

- **Mobile-First Development**: Start with the mobile layout and progressively enhance for larger screens.
- **Standard Breakpoints**: Use a consistent set of breakpoints across the app (mobile/tablet/desktop), and document them.
- **Design for Containers, Not Just Viewports**: Prefer layouts that adapt to available space (sidebars, cards, modals), not only screen width.
- **Fluid Layouts**: Use flexible grids and containers (percentages, flex, grid) that adapt naturally.
- **Relative Units**: Prefer rem/em for typography and spacing; avoid hard-coded pixel sizes except for hairlines/borders when needed.
- **Responsive Media**: Ensure images/video scale correctly (`max-width: 100%`, proper aspect ratios) and don’t overflow containers.
- **Content Priority**: Show the most important content first on smaller screens; defer or collapse secondary content.
- **Touch-Friendly Design**: Ensure tap targets are large enough (≈44x44px) and spaced to avoid accidental taps.
- **Don’t Rely on Hover**: Any hover-only affordance must have an equivalent for touch/keyboard users.
- **Readable Typography**: Maintain readable font sizes, line height, and line length across breakpoints without requiring zoom.
- **Safe Areas & Insets**: Account for notches/home indicators (safe-area insets) on modern mobile devices when relevant.
- **Performance on Mobile**: Optimize images/assets (responsive sizes, lazy-loading where appropriate); avoid heavy layout thrashing and large JS on initial load.
- **Test Across Devices**: Verify key screens across multiple widths and orientations; include edge cases (very small phones, large monitors, zoomed text).
- **Respect User Preferences**: Support reduced motion and high-contrast modes where applicable; avoid layout that breaks when text size is increased.
