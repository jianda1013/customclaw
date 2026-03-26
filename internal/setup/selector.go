package setup

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const maxVisible = 12

// interactiveSelect renders a scrollable list and lets the user navigate with
// ↑/↓ arrow keys and confirm with Enter.
//
// defaultIdx pre-selects the item at that index (pass -1 for none).
// Returns the selected item or ErrInterrupted on Ctrl+C / ESC.
func interactiveSelect(items []string, defaultIdx int) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select")
	}

	cursor := defaultIdx
	if cursor < 0 || cursor >= len(items) {
		cursor = 0
	}

	// Non-TTY (piped input, CI, tests): return the pre-selected item silently.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return items[cursor], nil
	}

	windowSize := min(len(items), maxVisible)
	totalLines := windowSize + 1 // list rows + hint row
	offset := 0

	// adjustOffset keeps the cursor inside the visible window.
	adjustOffset := func() {
		if cursor < offset {
			offset = cursor
		}
		if cursor >= offset+windowSize {
			offset = cursor - windowSize + 1
		}
	}

	hint := func() string {
		h := "  ↑↓ navigate · Enter select · Ctrl+C cancel"
		if len(items) > maxVisible {
			h += fmt.Sprintf("  [%d/%d]", cursor+1, len(items))
		}
		return h
	}

	// printInitial renders the list before raw mode (uses regular \n).
	printInitial := func() {
		adjustOffset()
		for i := range windowSize {
			item := items[offset+i]
			if offset+i == cursor {
				fmt.Printf("  \033[7m ▶ %-40s\033[0m\n", item)
			} else {
				fmt.Printf("     %-40s\n", item)
			}
		}
		fmt.Println(hint())
	}

	// redraw re-renders the list in raw mode (uses \r\n).
	redraw := func() {
		adjustOffset()
		fmt.Printf("\033[%dA", totalLines) // move cursor to top of list
		for i := range windowSize {
			item := items[offset+i]
			fmt.Printf("\r\033[2K") // clear line
			if offset+i == cursor {
				fmt.Printf("  \033[7m ▶ %-40s\033[0m\r\n", item)
			} else {
				fmt.Printf("     %-40s\r\n", item)
			}
		}
		fmt.Printf("\r\033[2K%s\r\n", hint())
	}

	// clearAll erases all rendered lines (called after restoring terminal).
	clearAll := func() {
		fmt.Printf("\033[%dA", totalLines)
		for range totalLines {
			fmt.Printf("\r\033[2K\n")
		}
		fmt.Printf("\033[%dA", totalLines)
	}

	printInitial()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Raw mode unavailable — return pre-selected item as-is.
		return items[cursor], nil
	}

	buf := make([]byte, 4)
	for {
		n, _ := os.Stdin.Read(buf)
		prev := cursor

		switch {
		case n == 1 && (buf[0] == '\r' || buf[0] == '\n'): // Enter
			term.Restore(int(os.Stdin.Fd()), oldState)
			clearAll()
			fmt.Printf("  %s\n", items[cursor])
			return items[cursor], nil

		case n == 1 && buf[0] == '\x03': // Ctrl+C
			term.Restore(int(os.Stdin.Fd()), oldState)
			clearAll()
			return "", ErrInterrupted

		case n == 1 && buf[0] == '\x1b': // lone ESC
			term.Restore(int(os.Stdin.Fd()), oldState)
			clearAll()
			return "", ErrInterrupted

		case n >= 3 && buf[0] == '\x1b' && buf[1] == '[':
			switch buf[2] {
			case 'A': // ↑
				if cursor > 0 {
					cursor--
				}
			case 'B': // ↓
				if cursor < len(items)-1 {
					cursor++
				}
			}
		}

		if cursor != prev {
			redraw()
		}
	}
}

// interactiveMultiSelect renders a checkbox list where Space toggles items
// and Enter confirms the selection.
// preSelected contains the items that should start checked.
// Returns the selected subset or ErrInterrupted on Ctrl+C / ESC.
func interactiveMultiSelect(items []string, preSelected []string) ([]string, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to select")
	}

	// Build initial checked state from preSelected.
	checked := make([]bool, len(items))
	preSet := make(map[string]bool, len(preSelected))
	for _, s := range preSelected {
		preSet[s] = true
	}
	for i, item := range items {
		checked[i] = preSet[item]
	}

	cursor := 0
	windowSize := min(len(items), maxVisible)
	totalLines := windowSize + 1 // rows + hint
	offset := 0

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-TTY: return preSelected as-is.
		return preSelected, nil
	}

	adjustOffset := func() {
		if cursor < offset {
			offset = cursor
		}
		if cursor >= offset+windowSize {
			offset = cursor - windowSize + 1
		}
	}

	hint := func() string {
		h := "  Space toggle · Enter confirm · Ctrl+C cancel"
		if len(items) > maxVisible {
			h += fmt.Sprintf("  [%d/%d]", cursor+1, len(items))
		}
		return h
	}

	renderRow := func(i int, raw bool) {
		item := items[offset+i]
		absIdx := offset + i
		box := "[ ]"
		if checked[absIdx] {
			box = "[x]"
		}
		nl := "\n"
		if raw {
			nl = "\r\n"
		}
		if absIdx == cursor {
			fmt.Printf("  \033[7m %s %-38s\033[0m%s", box, item, nl)
		} else {
			fmt.Printf("    %s %-38s%s", box, item, nl)
		}
	}

	printInitial := func() {
		adjustOffset()
		for i := range windowSize {
			renderRow(i, false)
		}
		fmt.Println(hint())
	}

	redraw := func() {
		adjustOffset()
		fmt.Printf("\033[%dA", totalLines)
		for i := range windowSize {
			fmt.Printf("\r\033[2K")
			renderRow(i, true)
		}
		fmt.Printf("\r\033[2K%s\r\n", hint())
	}

	clearAll := func() {
		fmt.Printf("\033[%dA", totalLines)
		for range totalLines {
			fmt.Printf("\r\033[2K\n")
		}
		fmt.Printf("\033[%dA", totalLines)
	}

	printInitial()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return preSelected, nil
	}

	buf := make([]byte, 4)
	for {
		n, _ := os.Stdin.Read(buf)
		prev := cursor
		toggled := false

		switch {
		case n == 1 && (buf[0] == '\r' || buf[0] == '\n'): // Enter — confirm
			term.Restore(int(os.Stdin.Fd()), oldState)
			clearAll()
			var result []string
			for i, item := range items {
				if checked[i] {
					result = append(result, item)
				}
			}
			return result, nil

		case n == 1 && buf[0] == ' ': // Space — toggle
			checked[cursor] = !checked[cursor]
			toggled = true

		case n == 1 && (buf[0] == '\x03' || buf[0] == '\x1b'): // Ctrl+C / ESC
			term.Restore(int(os.Stdin.Fd()), oldState)
			clearAll()
			return nil, ErrInterrupted

		case n >= 3 && buf[0] == '\x1b' && buf[1] == '[':
			switch buf[2] {
			case 'A':
				if cursor > 0 {
					cursor--
				}
			case 'B':
				if cursor < len(items)-1 {
					cursor++
				}
			}
		}

		if cursor != prev || toggled {
			redraw()
		}
	}
}
