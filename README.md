# Hless

# High level design
Every line in the input is broken up into segments. Each segment is
expected to fit into screen.

When moving the cursor down, once it reaches the bottom line, we move
down one segment at a time. I.e., it is possible that when pressing
down cursor will stay on the same line. Instead we will move the top
segment.