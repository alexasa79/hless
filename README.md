# Hless

# High level design
Every line in the input is broken up into segments. Each segment is
expected to fit into screen.

When moving the cursor down, once it reaches the bottom line, we move
down one segment at a time. I.e., it is possible that when pressing
down cursor will stay on the same line. Instead we will move the top
segment.

# TODO List
- [ ] Change the executable name to `hh`
- [ ] When interrupted when searching or going to the end of the
      document, jump to the last found line instead of staying in the
      same spot.
- [ ] Monitor the time it takes to render the screen and do not append
      new requests to redraw the screen if there is already a pending
      request to do that.
