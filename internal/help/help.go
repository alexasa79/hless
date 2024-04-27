package help

var Text =
`
# Hless - Pager for human beings

## Basic navigation

Press 'q' to leave this screen and up and down arrows to navigate. Below is
the list of supported keyboard shortcuts.

| Shortcut | Description |
| --- | --- |
| q or ctrl+c | Exit hless. |
| Up or k | Move the cursor one line up. |
| Down or j | Move the cursor one line down. |
| g | Move the cursor to the beginning of the file. |
| G | Move the cursor to the end of the file. |
| PgUp | Move the cursor one page up. |
| PgDown | Move the cursor one page down. |
| b | Move the cursor half a page up. |
| f | Move the cursor half a page down. |
| Enter | Move the screen to place the current line in the middle of the screen. |
| m | Toggle mark on the current line. |
| . | Jump to the next marked line. |
| , | Jump to the previous marked line. |
| h | Show this screen. |

## Search

Search in hless is different than in vi or less. Hless supports multiple
search patterns, up to 9. The typical n and N search for the last
search pattern.

| Shortcut | Description |
| --- | --- |
| n | Search down for the current search pattern. |
| N | Search up for the current search pattern. |
| / | Will start new search pattern. After entering the search pattern, search down. |
| ? | Start new search pattern. After entering the pattern, search up. |
| 1 through 9 | Search down for the respective search pattern. When pressed with shift, will search up instead. |
| e | Edit the current search pattern. |
| F2 | Switch to the previous search pattern. |
| F4 | Switch to the next search pattern. |
| Left | Select the previous word on the current line. |
| Right | Select the next word on the current line. |

`