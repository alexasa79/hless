package help

var Text = `# Hless - Pager for human beings

## Basic navigation

Press 'q' to leave this screen and up and down arrows to navigate. Below is
the list of supported keyboard shortcuts.

q or ctrl+c - Exit hless.
Up or k     - Move the cursor one line up.
Down or j   - Move the cursor one line down.
g           - Move the cursor to the beginning of the file.
G           - Move the cursor to the end of the file.
R           - Refresh the contents of the file.
PgUp        - Move the cursor one page up.
PgDown      - Move the cursor one page down.
b           - Move the cursor half a page up.
f           - Move the cursor half a page down.
Enter       - Move the screen to place the current line in the middle of the
              screen.
m           - Toggle mark on the current line.
.           - Jump to the next marked line.
,           - Jump to the previous marked line.
h           - Show this screen.

## Search

Search in hless is different than in vi or less. Hless supports multiple
search patterns, up to 9. The typical n and N search for the last search
pattern. / and ? will start a new search with a new pattern. Every search
pattern match is shown with a different color.

n           - Search down for the current search pattern.
N           - Search up for the current search pattern.
/           - Will start new search pattern. After entering the search
              pattern, search down.
?           - Start new search pattern. After entering the pattern, search up.
1-9         - Search down for the respective search pattern.
1-9 + Shift - Search for the respetive search pattern.
e           - Edit the current search pattern.
F2 or [     - Switch to the previous search pattern.
F4 or ]     - Switch to the next search pattern.
Left        - Select the previous word on the current line.
Right       - Select the next word on the current line.
a           - Start new search pattern with the selected word. Will NOT search
              for the next match.
A           - Add selected word to the current search pattern with \|\. Will
              NOT search for the next match.

## Show and hide portions of the file

Hiding various portions of the file is available through 'v' keyboard
shortcut. To enter one of the shortcuts below, first press 'v' and then the
shortcut.

a - Hide everything after the current line.
b - Hide everything before the current line.
s - Only show lines matching the specified pattern.
h - Hide the lines matching the specified pattern.
c - Unhide everything. Show everything.
A - Add currently selected word to show pattern.

`
