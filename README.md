# Hless
Hless is a terminal pager program (like `less` or `more`) designed to aid at
analyzing log files.

# High level design
Every line in the input is broken up into segments. Each segment is
expected to fit into screen.

When moving the cursor down, once it reaches the bottom line, we move
down one segment at a time. I.e., it is possible that when pressing
down cursor will stay on the same line. Instead we will move the top
segment.

## Representation of the contents of the file
The contents of the file is served by a `Lines` module. Contents of the file
is broken up into multiple `Line` objects. Each `Line` has a name, an offset
and its contents.

`Section` object represents a named portion of the file. It is part of a
mechanism used to generate names for lines. For example, the very first
section will start at offset 0 and will an empty name. When going to the end
of the file, unless we already have a section to represent it, I will create a
new section called `end`. Lines in this section will be called "end-1",
"end-2", etc.

# TODO List
- [ ] Change the executable name to `hh`
- [ ] When interrupted when searching or going to the end of the
      document, jump to the last found line instead of staying in the
      same spot.
- [ ] Monitor the time it takes to render the screen and do not append
      new requests to redraw the screen if there is already a pending
      request to do that.
