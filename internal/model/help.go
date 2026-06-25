package model

func (m *Model) renderHelp() string {
	return `
 ` + m.textHeader1Style.Render("# Hless - Pager for human beings") + `

 ` + m.textHeader2Style.Render("## Basic navigation") + `

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
 m or space  - Toggle mark on the current line. Press again to cycle through
               different colors.
 M           - Clear all marks in the current line.
 .           - Jump to the next marked line.
 ,           - Jump to the previous marked line.
 i           - Show inventory.
 h           - Show this screen.

 ` + m.textHeader2Style.Render("## Search") + `

 Search in hless is different than in vi or less. Hless supports multiple
 search patterns, up to 9. The typical n and N search for the last search
 pattern. / and ? will start a new search with a new pattern. Every search
 pattern match is shown with a different color.

 n             - Search down for the current search pattern.
 N             - Search up for the current search pattern.
 /             - Will start new search pattern. After entering the search
                 pattern, search down.
 ?             - Start new search pattern. After entering the pattern, search
                 up.
 1-9           - Search down for the respective search pattern.
 1-9 + Shift   - Search for the respetive search pattern.
 e             - Edit the current search pattern.
 F2 or [       - Switch to the previous search pattern.
 F4 or ]       - Switch to the next search pattern.
 Left          - Select the previous word on the current line.
 Left + Shift  - Expand selection left to the next word.
 Right         - Select the next word on the current line.
 Right + Shift - Expand selection right to the next word.
 a             - Start new search pattern with the selected word. Will NOT
                 search for the next match.
 A             - Add selected word to the current search pattern with \|\.
                 Will NOT search for the next match.

 ` + m.textHeader2Style.Render("## Show and hide portions of the file") + `

 Hiding various portions of the file is available through 'v' keyboard
 shortcut. To enter one of the shortcuts below, first press 'v' and then the
 shortcut. Comands below change the alternate view.

 a     - Hide everything after the current line.
 b     - Hide everything before the current line.
 s     - Only show lines matching the specified pattern.
 h     - Hide the lines matching the specified pattern.
 c     - Unhide everything. Show everything.
 A     - Add currently selected word to show pattern.
 v     - Toggle between showing everything and the alternate view.
 Space - Add current line to the alternate view.

 ` + m.textHeader2Style.Render("## Zero timestamp") + `

 Relative timestamp display is available through the '0' keyboard shortcut.
 To enter one of the shortcuts below, first press '0' and then the shortcut.

 s - Show relative timestamps using the current line as zero.
 h - Hide relative timestamps.
 0 - Go to the zero timestamp line.


`
}
