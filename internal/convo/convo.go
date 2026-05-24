package convo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"

	"github.com/xmd-scripts/xmd/internal/tool"
)

// Read reads all messages from a JSONL context file.
// Returns an empty slice if the file does not exist.
func Read(path string) ([]tool.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var msgs []tool.Message
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var m tool.Message
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, scanner.Err()
}

// Append appends messages to a JSONL context file, creating it if needed.
func Append(path string, msgs []tool.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, m := range msgs {
		if err := enc.Encode(m); err != nil {
			return err
		}
	}
	return nil
}
