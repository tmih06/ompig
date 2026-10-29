# Stock Ompig and Ompig Standard

Stock Ompig contains the Pi-compatible coding agent and generic support for
extensions, Packages, Ompiglets, Ompiglet Binaries, and local reference documents.
It does not activate product extensions. Its startup header shows the Ompig ompig head
(D2), and its built-in `ompig-login` extension registers `/sprite`.

## Stock capabilities

| Capability | Purpose |
|---|---|
| `ompig docs` | Materialize and read the documentation that matches the binary. |
| `ompig extension ...` | Author, inspect, validate, and run ordinary extensions. |
| `ompig install` | Install Packages that distribute Resources. |
| `ompig ompiglet ...` | Validate, register, inspect, and build Ompiglets. |
| Ompiglet Binary verification | Verify the embedded Ompiglet and component closure before startup. |

These capabilities do not select a product composition.

## Ompig Standard

Ompig Standard is an explicit Ompiglet under `ompiglets/standard/`. It selects
ordinary extension Resources through public Ompiglet contracts. Bare `ompig` does
not load those Resources.

Run the source composition from a Ompig source checkout:

```bash
ompig --ompiglet ompiglets/standard/ompig-standard.yaml
```

Build a directly executable composition:

```bash
ompig ompiglet build ompiglets/standard/ompig-standard.yaml \
  --format binary \
  --out ./ompig-standard
```

Installing a Package can make Standard Resources available. It does not create,
select, or activate the Ompig Standard Ompiglet.

## Login identity

Stock Ompig's startup header shows the sprite's 7-line pixel ompig, with the version and hints beside it, where Pi shows its logo (D2). The built-in `ompig-login` extension registers `/sprite`, which chooses one of fifteen built-in sprites or a sprite an extension registered with `ui.registerSprite`, and saves the choice in `$OMPIG_HOME/state/ompig-standard/login.json`. Use `/sprite` to open a picker, `/sprite list` and `/sprite set <id>` for text-based control, and `/sprite preview [id]` to see a sprite's full art.

Ompig Standard selects its own `piglogin` extension. That extension installs its
login through `ui.setLogin` and replaces the built-in `/sprite` command.

## Ompig Runner

Ompig Standard selects the `pigrunner` extension. Use `/runner` or `/ompig-runner`
to open it. The extension owns timer-driven custom UI and stores the high score
at `<config-root>/state/ompig-standard/pigrunner.json`.

The host gives one focused extension exclusive terminal input. Timer redraws
use the generic invalidation callback and do not block the TUI render loop.

## Additional batteries

Each additional battery must be an ordinary fuse-compatible Go extension
Resource selected by the Standard Ompiglet. It must work as a source subprocess
extension and as a fused component in the Ompig Standard Binary. The Standard
build fails instead of using a subprocess fallback. Stock Ompig must not import or
activate it.
