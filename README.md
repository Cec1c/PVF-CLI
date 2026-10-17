<div align="center">

# PVF-CLI

![Status: controlled editing](https://img.shields.io/static/v1?label=status&message=controlled%20editing&color=2563EB&style=flat-square)
![Go](https://img.shields.io/static/v1?label=Go&message=1.26.2&color=00ADD8&style=flat-square&logo=go&logoColor=white)

A standalone command-line toolkit for inspecting, searching, exporting and editing DNF PVF archives, including raw bytes and token JSON.

[简体中文](README.zh-CN.md) · [CLI reference](docs/CLI.md) · [Patch plans](docs/PATCH.md) · [Agent entry](AGENTS.md)

</div>

> [!NOTE]
> `info / list / read / search / export / patch / verify` are implemented. `patch` creates a separate candidate from an explicit plan; `verify` checks the planned changes and preservation of every other entry. Input files are never overwritten.

## Scope

Supports NKPI and protected NKPI. Read entries as display text, raw bytes or JSON preserving token kinds and payload bits. Search full text or a named section such as `[name]`. Searches process one decompressed group at a time without retaining the decoded text cache.

Runs without pvfUtility/PUT, a GUI, a database or a game server. Export rejects existing files, path traversal and escaping symlinks. Other PVF variants require separate samples and validation.

## Quick start

Requires Go 1.26.2; the initial build downloads `golang.org/x/text v0.36.0`. On Windows, run `BUILD.bat` to test and build with a project-local cache. The executable is written to `dist/pvf.exe`.

```text
dist\pvf.exe info --pvf C:\samples\Script.pvf --json
dist\pvf.exe search --pvf C:\samples\Script.pvf --prefix stackable --section name --query 小晶块 --json
dist\pvf.exe read --pvf C:\samples\Script.pvf --path "itemshop/(r)equipmentshop7.shp" --format tokens --out shop.tokens.json
dist\pvf.exe export --pvf C:\samples\Script.pvf --path npc/seria.npc --out extracted
```

An output file must not already exist. An export directory may exist, but its files will not be overwritten. See the [CLI reference](docs/CLI.md) for options, exit codes and format boundaries.

## Controlled editing

Read the target entry as raw bytes or tokens, then fill in the input and entry hashes in a [patch plan](docs/PATCH.md):

```text
dist\pvf.exe patch --pvf baseline.pvf --plan approved.json --out candidate.pvf
dist\pvf.exe verify --pvf baseline.pvf --plan approved.json --out candidate.pvf
```

Schema 1 supports raw replacement. Schema 2 supports protected NKPI token edits and appended strings. The CLI does not add new paths or compile display text back into scripts.

## Start development

1. Read [AGENTS.md](AGENTS.md) and [the handoff](docs/HANDOFF.md).
2. Review [the source map](reference/README.md) and [source manifest](reference/source-manifest.json).
3. Validate with the included synthetic fixtures; real samples require an explicit `PVFCLI_REAL_PVF_SMOKE` selection and are not distributed.
4. Follow [the implementation plan](docs/IMPLEMENTATION_PLAN.md) and [validation criteria](docs/VALIDATION.md).

The standalone module is `pvfcli`, with its core in `internal/pvf`. Synthetic test fixtures are included; real game assets remain ignored. Windows builds have been validated; runtime behavior on other target platforms requires separate validation.

## Existing foundation

The inspected source contains roughly 2,100 lines of PVF core implementation and 1,800 lines across five scratch commands, excluding tests and the document parser. The core's only observed external import is `golang.org/x/text/encoding/simplifiedchinese`; the full server dependency list is unnecessary for a standalone CLI.

> [!IMPORTANT]
> Display text rounds floats and renders some token types as placeholders. Use raw bytes or token JSON as the basis for editing. Opening an archive still loads the whole archive into memory; grouped searches are not disk-streamed archive loading. See the [validation criteria](docs/VALIDATION.md) for the verification scope.

## References and origin

See [external projects](docs/REFERENCES.md) for related GitHub tools and their verified limits. The core originates from [ServerS5A3 / DNF90](https://github.com/Cec1c/ServerS5A3), with the source commit and per-file SHA256 recorded in the manifest. No license has been assigned to this project, and this repository grants no new relicensing rights to the source code.
