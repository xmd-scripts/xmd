package tool

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var (
	stdinReader     *bufio.Reader
	stdinReaderOnce = &sync.Once{}
)

func stdinLine() (string, error) {
	stdinReaderOnce.Do(func() { stdinReader = bufio.NewReader(os.Stdin) })
	line, err := stdinReader.ReadString('\n')
	return line[:len(line)-len("\n")], err
}

type questionArgs struct {
	Prompt string `json:"prompt"`
}

var openTTY = func() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

// Question asks the user a question interactively via /dev/tty.
func Question(argsJSON string) (string, error) {
	var args questionArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("question: invalid arguments: %w", err)
	}
	if args.Prompt == "" {
		return "", fmt.Errorf("question: prompt is required")
	}

	tty, err := openTTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n> ", args.Prompt)
		return stdinLine()
	}
	defer tty.Close()

	fmt.Fprintf(tty, "\n%s\n> ", args.Prompt)

	scanner := bufio.NewScanner(tty)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", fmt.Errorf("question: failed to read input from tty")
}
