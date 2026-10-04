# fzf-multicolumn

A fork of [junegunn/fzf](https://github.com/junegunn/fzf) that adds **`--grid=COLS`** — a multi-column grid layout with two-dimensional navigation and dynamic column widths.

```
╭────────────────────────────────────────────────────────────╮
│ ▌ query_                                                   │
│   24/24 ─────────────────────────────────────────────────  │
│ ▌ claude    ▌  yeowoolmac  ▌  Superhuman                 │
│ ▌ codex     ▌  grimoire    ▌  stage0                     │
│ ▌ opencode  ▌  oci-ubuntu  ▌  repo-template              │
│ ▌ hermes    ▌  bingus      ▌  Documents                  │
│ ▌ grok      ▌               ▌  Photopeace                 │
╰────────────────────────────────────────────────────────────╯
```

Upstream fzf has no grid layout, and the maintainer has [declined it](https://github.com/junegunn/fzf/issues/4006) as requiring changes too invasive for the mainline codebase. Fair enough — this fork carries the feature instead. Everything else is stock fzf; the grid code is an isolated rendering path (~280 lines) to keep rebases onto new upstream releases painless.

## Usage

```sh
seq 100 | fzf-multicolumn --grid=4
printf 'one\ntwo\nthree\nfour\nfive\nsix\n' | fzf-multicolumn --grid=3 --height=6 --reverse --cycle
```

### New options

| Option | Description |
| ------ | ----------- |
| `--grid=COLS` | Lay items out in a row-major grid of COLS columns (`--grid=1` and `--grid=0` mean a normal list) |
| `--no-grid` | Disable grid mode |
| `--grid-gap=COLS` | Minimum number of spaces between grid columns (default: 2) |
| `--grid-span-prefix=PREFIX` | Opt in to per-record track spans encoded as `PREFIX`*N*`PREFIX`*payload* |

### New actions

| Action | Default binding | Description |
| ------ | --------------- | ----------- |
| `grid-left` | `left` (when `--grid` is set) | Move to the previous item within the row |
| `grid-right` | `right` (when `--grid` is set) | Move to the next item within the row |

`ctrl-b` / `ctrl-f` still move the cursor inside the query, and explicit `--bind` definitions override the grid defaults.

### Spanning grid tracks

Spans are opt-in, so existing input is unchanged unless a prefix is configured.
For example, `--grid=6 --grid-span-prefix=@@` interprets `@@5@@details` as
`details` occupying five consecutive tracks:

```sh
printf '%s\n' '[ ]' '@@5@@Module details' '[*]' '@@5@@Another module' |
  fzf-multicolumn --grid=6 --grid-span-prefix=@@
```

The marker is removed before ANSI processing, field transforms, matching,
preview placeholders, and output (including `--filter`, `--accept-nth`,
`--read0`, and reload input). Complete markers require a positive decimal span;
span zero, overflow, and spans wider than `--grid` are errors. The prefix must
be nonempty and cannot begin with an ASCII digit. Incomplete or
otherwise malformed markers remain literal. Items are placed left-to-right in
input order and move to the next row when their complete span does not fit.

### Behavior

- **Dynamic track widths** — span-1 cells establish per-track widths, while spanning cells impose aggregate requirements across all covered tracks. When the grid does not fit, tracks shrink without splitting a cell's decorations; extremely narrow terminals collapse whole tracks. Lists over 4096 items skip text measurement but retain the same renderability floors and collapse rules.
- **Geometric navigation** — `left`/`right` traverse selectable cells in the current logical row. `up`/`down` choose an overlapping cell in the adjacent row, or the nearest cell when none overlaps; with `--cycle`, vertical movement wraps between logical rows. Page and offset movement are measured in logical rows. Jump labels are assigned only to visible selectable placements, skipping inert placeholders.
- **Live reflow** — typing a query reflows the matched items through the grid, with per-character match highlighting inside each cell.
- **Row-aligned scrolling** — the scroll offset is kept row-aligned and the scrollbar tracks rows.
- **Placeholder cells** — whitespace-only items are treated as blank padding: they render as empty space, the cursor skips over them, and mouse clicks on them are ignored. This is what makes semantic columns (below) work — short columns are padded with `' '` items that can never be focused or selected.
- Multi-select markers, `--layout=default|reverse`, `--header`, `--border`, `--height`, and preview windows all work as usual.

### Semantic columns

The grid is row-major, so fixed-meaning columns (categories) are a matter of interleaving your input and padding short categories with blank items:

```zsh
# col 1 = tools, col 2 = hosts, col 3 = folders
local -a rows
for (( i = 1; i <= max; i++ )); do
    rows+=("${tools[i]:- }" "${hosts[i]:- }" "${folders[i]:- }")
done
choice=$(fzf-multicolumn --grid=3 --height=$(( max + 5 )) --reverse <<< "${(F)rows}")
```

### Limitations

Grid cells are single-line: only the first line of a multi-line item is shown, and `--wrap` and `--gap` are not grid-aware. If you need those, you probably want the regular list layout anyway.

## Installation

Prebuilt static binaries for macOS (arm64/amd64) and Linux (amd64/arm64) are on the [releases page](https://github.com/LPFchan/fzf-multicolumn/releases), with a SHA-256 checksums file per release:

```sh
curl -fsSLO https://github.com/LPFchan/fzf-multicolumn/releases/download/v0.74.0-multicolumn.3/fzf-multicolumn-0.74.0-multicolumn.3-darwin_arm64.tgz
tar -xzf fzf-multicolumn-*.tgz && install fzf-multicolumn ~/.local/bin/
```

Or build from source (Go 1.23+):

```sh
git clone https://github.com/LPFchan/fzf-multicolumn.git
cd fzf-multicolumn
go build -o fzf-multicolumn .
install fzf-multicolumn ~/.local/bin/   # or /opt/homebrew/bin on macOS
```

> **macOS note:** when reinstalling over an existing copy, `rm` the old binary before `cp`-ing the new one. Overwriting a signed binary in place invalidates the kernel's per-vnode code-signature cache and the next launch gets SIGKILLed with no error message.

The binary is a drop-in superset of the fzf release it's based on, but it's built and versioned independently — it does not replace your packaged `fzf`, and the shell integration scripts still belong to upstream.

The checkout installers preserve an existing grid-capable `bin/fzf` binary.
The Bash installer downloads this fork’s macOS/Linux release assets when needed
and builds this checkout on other platforms. The PowerShell installer builds
this checkout with Go; it requires Go on Windows.

## Tracking upstream

The fork lives on the `multicolumn` branch, based on an upstream release tag. To move to a new fzf release:

```sh
git fetch origin --tags
git rebase <new-tag> multicolumn
(cd src && go run golang.org/x/tools/cmd/stringer@latest -type=actionType)  # regenerate action names
go build -o fzf-multicolumn . && go test ./src/...
```

The grid changes are concentrated in `src/terminal.go` (rendering, constrain, movement) and `src/options.go` (`--grid` parsing and the arrow-key rebind), so conflicts are rare.

## License

[MIT](LICENSE), same as upstream. All credit for fzf itself goes to [Junegunn Choi](https://github.com/junegunn) — consider [sponsoring him](https://github.com/sponsors/junegunn).
