/*
Copyright © 2024 IAN R FERGUSON IANFERGUSONRVA@gmail.com
*/
package asana

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ABOUT - Accepts YYYY-MM-DD, "today" or "tomorrow"; empty input means no date
func parseDueDate(raw string) (string, error) {
	const layout = "2006-01-02"
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return "", nil
	case "today":
		return time.Now().Format(layout), nil
	case "tomorrow":
		return time.Now().AddDate(0, 0, 1).Format(layout), nil
	}
	if _, err := time.Parse(layout, raw); err != nil {
		return "", fmt.Errorf("%q is not a valid date, use YYYY-MM-DD", raw)
	}
	return raw, nil
}

func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func promptLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// ABOUT - Reads input without echoing it when attached to a terminal
func promptSecret(reader *bufio.Reader, prompt string) (string, error) {
	if !stdinIsTerminal() {
		return promptLine(reader, prompt)
	}
	fmt.Print(prompt)
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(secret)), nil
}
