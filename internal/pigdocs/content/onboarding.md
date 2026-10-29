# Get started with Ompig

Stock Ompig includes the reference documentation required to configure models,
author extensions, use Packages, and compose Ompiglets. It does not register an
onboarding extension.

On a first interactive start, Ompig shows a first-time setup before the session opens (D88). It appears when `OMPIG_CODING_AGENT_DIR` is not set and the agent directory has no `settings.json` yet, which includes every fresh `OMPIG_HOME`. It has three steps: a theme (System, Dark or Light, with a live preview), a sprite (the ompig the header shows) and a choice to share anonymous usage data. Finishing saves the answers to `settings.json` and to the sprite state, so the setup does not return. `Esc` skips it and saves no answer. The setup still does not return: the same start writes `settings.json` (with `lastChangelogVersion`), and the setup appears only while that file is missing. Print, JSON and RPC modes never show it. Pi 1.4 shows its own two-step version (theme and analytics) only for its official distribution with `PI_EXPERIMENTAL=1` (`startup-ui.ts` `shouldRunFirstTimeSetup`), so Ompig shows it more often than Pi does. Change the theme later with `/settings`, the sprite with `/sprite`, and the usage-data choice by setting `enableAnalytics` in `settings.json`.

Ask the running agent a direct setup question when a model is available. Tell it
to read the matching local documentation before it changes files.

## Set up a project Ompiglet

Use this prompt when you want Ompig to create a project-specific composition:

```text
Read the local Ompig documentation. Set up a Ompiglet for this repository. Explain
each selected model preference, system prompt, built-in tool, skill, and
extension. Write the Ompiglet source, validate it with ompig ompiglet validate, and
show the exact ompig --ompiglet command.
```

The expected result is a Ompiglet source under `~/.ompig/ompiglets/`, a project-local
`.ompig/ompiglets/` directory, or another explicit path.

## Scaffold an extension

```bash
ompig extension init ./my-extension --lang go
ompig install ./my-extension --validate-only --json
ompig -e ./my-extension
```

Edit the extension and use `/reload` in the session. Read `extensions.md` for
the source forms. Read `extension-api.md` for the host API.

## Use Ompig Standard

Ompig Standard is an explicit Ompiglet composition. It is not part of bare Stock
Ompig. From a Ompig source checkout, run:

```bash
ompig --ompiglet ompiglets/standard/ompig-standard.yaml
```

Build a directly executable composition when you do not want the target host to
compile compatible Go or Rust extension sources:

```bash
ompig ompiglet build ompiglets/standard/ompig-standard.yaml \
  --format binary \
  --out ./ompig-standard
```

## Related documents

- [Concepts](concepts.md)
- [Extensions](extensions.md)
- [Extension API](extension-api.md)
- [Packages](packages.md)
- [Ompiglets](ompiglets.md)
- [Stock Ompig and Ompig Standard](built-in-extensions.md)
