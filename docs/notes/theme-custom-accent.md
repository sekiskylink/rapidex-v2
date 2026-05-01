# Theme Custom Accent

## Summary

Both `web/` and `desktop/frontend/` now support a `custom` appearance preset that is driven by a locally persisted accent color. The active theme still resolves entirely on the client, with no backend settings dependency.

## Implementation Notes

- Curated presets remain available and were expanded with additional options.
- Selecting a custom accent automatically switches the active preset to `custom`.
- The stored preference keeps:
  - the selected preset id
  - an optional normalized hex accent color
- Theme generation derives secondary/background values from the chosen accent so light and dark modes stay visually coherent without exposing a full manual theme editor.

## Persistence

- Web stores the custom accent in the existing UI preferences localStorage payload.
- Desktop stores the same field through the Wails settings store under `uiPrefs`.
- Invalid stored custom colors are sanitized away during load, and the clients fall back to the default custom accent when rendering the custom preset.
