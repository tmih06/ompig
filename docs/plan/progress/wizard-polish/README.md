# wizard-polish: first-time setup against Pi 1.0.0

Captures: VHS (xterm.js), 1000x760 px, 14 px font, a fresh `HOME` per run, the same `/tmp/wp/work` cwd. `pi-*` is Pi 1.0.0 with `PI_EXPERIMENTAL=1` (its setup gate), `ompig-before-*` is agg-100 merged with rev-first-run-sprite (`9a95f9f09`), `ompig-after-*` is this branch. Keys: theme step, Down Down (light), Up (dark), Enter, then on Ompig Down x11 (Kratos) and Enter, then Enter (finish).

| Item | Before | After | Pi 1.0.0 |
|---|---|---|---|
| 1 sprite saved / shown | `ompig-before-4-sprite-kratos.png`, `ompig-before-5-analytics.png` (analytics step drew the active sprite, not the chosen one) | `ompig-after-4-sprite-kratos.png`, `ompig-after-5-analytics.png`, `ompig-after-6-after-setup.png` (Kratos through every step and in the header; `login.json` = `kratos`) | n/a (Ompig-only step, D88) |
| 2 placement | dialog inside the TUI under the header: two pigs, header clipped, bottom border cut | own startup screen before the TUI, one ompig | `pi-1-theme-system.png` (own screen) |
| 3 gap after setup | `ompig-before-6-after-setup.png` | `ompig-after-6-after-setup.png` | `pi-6-after-setup.png` |
| 4 theme preview | `ompig-before-1..3` | `ompig-after-1..3` | `pi-1..3` |
| 5 naming | `Welcome to ompig`, `Agentic Ompig` | `Welcome to Ompig`, `Ompig: The minimal coding agent, in Go.` | `Welcome to pi` |
| 6 create your own | none | last sprite-step row `Create your own...` (`ompig-after-4`, row 16/16), `/sprite create` | n/a |

Findings:

- Item 1: in a 100x40 tmux the before binary saved `kratos` for Kratos, but the analytics step drew the active sprite (the default) instead of the chosen one, and the dialog (34 rows plus the header) overflowed the screen, so the editor slot repainted over stale rows. Both are gone: the dialog's ompig is the highlighted sprite through the last step, the list scrolls in eight rows, and the dialog runs on its own screen.
- Item 3: Pi 1.0.0's default `tuiMode` is fullscreen; its editor and footer sit on the last rows with the blank viewport between the transcript and the editor (`pi-6-after-setup.png`). Ompig's after-setup screen has the same rows: the header slot reserves none. The large blank area under the header during setup came from the in-TUI dialog; it is gone.
- Item 4: the theme step's ANSI bytes (tmux `capture-pane -e`, 100x40) equal Pi's for System, Dark and Light except the logo rows and `pi`/`Ompig`. System resolves the same way in both (same OSC 11 reply).
- The hint line differed: Ompig drew `Enter continue  Escape/Ctrl+C`, Pi draws `enter continue  escape/ctrl+c` (`keyHint` uses `keyText`, not `keyDisplayText`). Fixed.

`/sprite` picker after this change (tmux, 100x40, no credentials; the line above the picker is `/sprite create` from setup's `Create your own...`):

```text
 Designing a sprite needs a working model. Run /login to log into a provider, select a model with
 /model, then run /sprite create.
 Choose a Ompig sprite
 → Ompig: The minimal coding agent, in Go.
   Classic Ompig: Pink, polite, and patient.
   ...
   Sheriff Ompig: Laying down the law, one commit at a time.
   Create your own...
 ↑↓ navigate  enter select  escape/ctrl+c cancel
```

Gap check on an ordinary start (settings.json present, no setup), tmux, editor border rows (top/bottom) of Ompig and Pi 1.0.0 at the same size:

| Size | Ompig | Pi 1.0.0 |
|---|---|---|
| 80x24 | 20, 22 | 20, 22 |
| 100x40 | 36, 38 | 36, 38 |
| 160x50 | 46, 48 | 46, 48 |

The editor and footer occupy the same rows in both, and the blank area between the header and the editor is Pi's fullscreen layout. The header slot reserves no rows.

Owner decision 2026-10-02 11:57: `Create your own...` (setup sprite step and `/sprite` picker) shows how to create a sprite and starts no model turn; in setup it keeps the active sprite and continues. `/sprite create` runs the guided turn when a model with credentials exists and explains `/login` otherwise.
