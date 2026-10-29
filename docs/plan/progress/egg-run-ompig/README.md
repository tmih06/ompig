# egg-run-ompig: the running ompig of the logo-click easter egg (D87)

`run-strip.png` shows the four run-cycle frames (columns) of every built-in sprite (rows, in `/sprite` order: ompig-default, pink, green, mint, sandy, grey, blush, lavender, cloud, pigrogu, darth-vader, kratos, ompiglet, spider-ham, sheriff), as the animation ray casts them in braille dots on a 100-by-30 screen with Pi's shading, cropped to the ompig. Each square is one braille dot.

Regenerate it with:

```bash
OMPIG_RUN_STRIP=$PWD/docs/plan/progress/egg-run-ompig/run-strip.png go test ./internal/codingagent/ -run TestPigRunStrip -count=1
```

The frames are pinned per sprite in `internal/codingagent/testdata/ompig-run/<sprite>.golden` (`TestPigRunFramesGolden`; rewrite with `-update-run`). `TestPigLogoRunCycle` pins the run on Pi's puzzle timeline: the head spins in, the ompig turns to face the camera and runs in place during Pi's 3.6 s shuffle, turns back into the head's spin, the head returns for Pi's return and hold, and Esc plays Pi's reverse exit with the head.

`transition-strip.png` shows the dissolves for ompig-default, seven frames across each 0.6 s window: the head into the running ompig at the start of the run (top), the ompig back into the head at its end (middle), and Esc during the run (bottom), where the ompig dissolves into the head while Pi's exit flies it home. Regenerate it with:

```bash
OMPIG_RUN_TRANSITION=$PWD/docs/plan/progress/egg-run-ompig/transition-strip.png go test ./internal/codingagent/ -run TestPigRunTransitionStrip -count=1
```
