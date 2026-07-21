// DOM-level UI state queries. Keyboard handlers registered on window (Escape,
// tile shortcuts, Cmd-V) need to know when a modal layer is covering the
// grid so keystrokes don't leak through to it.

/** Whether a modal Radix dialog/alertdialog is currently open. */
export function modalOpen(): boolean {
  return (
    document.querySelector(
      '[role="dialog"][data-state="open"], [role="alertdialog"][data-state="open"]',
    ) != null
  );
}

/** Whether the settings view is covering the (still mounted) grid. */
export function settingsViewOpen(): boolean {
  return document.querySelector('[role="dialog"][aria-label="Settings"]') != null;
}

/** Whether grid-level keyboard shortcuts should stay quiet right now. */
export function gridInputBlocked(): boolean {
  return modalOpen() || settingsViewOpen();
}
