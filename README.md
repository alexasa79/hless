# Hless

Hless (`hh`) is an interactive terminal pager built for reading and analyzing
log files. It keeps the familiar, keyboard-driven feel of tools such as `less`,
while adding features that make it easier to follow several events at once,
preserve points of interest, isolate relevant lines, and reason about timing.

Hless is especially useful when you need to:

- trace several request IDs, services, threads, or error types through one log;
- compare events without repeatedly replacing a single search query;
- mark important lines and return to them later;
- temporarily hide noise or focus on a subset of the file;
- measure elapsed time between timestamped events; or
- inspect long lines without losing your place.

## Features

### Multiple search patterns

Hless can keep up to **nine search patterns active at the same time**. Each
pattern has its own color, so matches remain visually distinct throughout the
file. Searches can be plain text or regular expressions and can be navigated
independently in either direction.

This is useful for following related signals together—for example, a request
ID, a database warning, and a retry message—without combining everything into
one complicated expression or losing an earlier search.

- `/` creates a pattern and searches forward; `?` creates one and searches
  backward.
- `n` and `N` move through matches for the current pattern.
- `1`–`9` search forward using a specific pattern; their shifted forms search
  backward.
- `[` / `F2` and `]` / `F4` switch the current pattern.
- `e` edits the current pattern.
- `Tab` switches between plain-text and regular-expression matching while
  editing.
- `a` creates a search from the selected word; `A` adds the selection to the
  current regex as an alternative.

The status area shows the active searches, their colors, match counts, and
nearby matches. The currently selected search is emphasized.

### Colored line markers

Press `m` or `Space` to mark the current line. Repeated presses cycle through
five marker colors, which makes it possible to give landmarks different
meanings—for example, red for failures, green for recovery, and another color
for investigation notes.

Use `.` and `,` to jump to the next and previous marked lines. `M` clears the
mark on the current line. Marker summaries in the status area also show where
marked lines sit relative to the cursor.

### Timestamp awareness

Hless recognizes common ISO-style timestamps at the beginning of log lines,
including bracketed timestamps, fractional seconds, `Z`, and numeric timezone
offsets. Examples include:

```text
2026-07-15T14:37:50.721Z request completed
2026-07-15 14:37:50.721999-04:00 worker started
[2026-07-15T18-37-50Z] alternate separator format
```

The zero-timestamp mode turns the current line into a time origin and displays
the elapsed time beside other timestamped lines. This makes latency, gaps, and
ordering visible without copying timestamps into another tool. Hless preserves
the precision present in the source timestamp, and lines without a recognized
timestamp remain readable without a relative value.

Press `0`, then:

- `s` to use the current line as zero and show relative timestamps;
- `h` to hide relative timestamps; or
- `0` to return to the zero line.

### Focused alternate views

The alternate view lets you narrow a noisy log without changing the underlying
file. Press `v`, followed by:

- `s` to show only lines matching a regular expression;
- `h` to hide lines matching a regular expression;
- `b` to hide everything before the current line;
- `a` to hide everything after the current line;
- `Space` to add the current line to the alternate view;
- `A` to add the selected word to the show pattern;
- `v` to toggle between the full and alternate views; or
- `c` to clear the alternate view.

This works well for isolating a time window, removing repetitive health checks,
or building a compact view of related events.

### Log-friendly navigation

Long input lines are split into screen-sized visual segments. Moving through
the file scrolls these segments naturally, so wrapped records remain navigable
while still behaving as a single logical line. Hless also provides full-page
and half-page movement, start/end jumps, refresh support, line selection, and
an inventory screen that summarizes active searches and filters.

## Building

Hless requires Go 1.21 or newer.

```sh
git clone https://github.com/alexasa79/hless.git
cd hless
make hless
```

This produces the `hh` executable in the project directory. To run the test
suite:

```sh
make test
```

The default `make` target also builds Linux AMD64 and ARM64 binaries.

## Usage

Open a file:

```sh
./hh application.log
```

Or page piped input:

```sh
journalctl -u my-service | ./hh
```

```text
Usage: hh [OPTIONS] FILE

Options:
  -h, --help            Show command-line help
  -V, --version         Show the version
      --debug MODS      Enable debug logging for comma-separated modules
      --trace MODS      Enable trace logging; implies debug logging
      --profile         Write a CPU profile to hless.prof
```

Supported logging modules are `all`, `view`, `model`, `search`, `lines`, and
`io`.

## Essential keys

| Key | Action |
| --- | --- |
| `↑` / `k`, `↓` / `j` | Move by one logical line or visual segment |
| `PgUp`, `PgDown` | Move by one page |
| `b`, `f` | Move by half a page |
| `g`, `G` | Go to the beginning or end |
| `Enter` | Center the current line |
| `R` | Refresh the input |
| `/`, `?` | Create a forward or backward search |
| `n`, `N` | Next or previous match for the current search |
| `m` / `Space` | Add or cycle a colored marker |
| `.`, `,` | Next or previous marked line |
| `v` | Open alternate-view commands |
| `0` | Open relative-timestamp commands |
| `i` | Show searches, filters, and other active state |
| `h` | Show the complete built-in help |
| `q` / `Ctrl-C` | Quit |

The built-in help (`h`) is the authoritative reference for all interactive
shortcuts.
