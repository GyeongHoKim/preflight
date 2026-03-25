package repotools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

// ReadArgs is the JSON shape for read_file.
type ReadArgs struct {
	Path     string `json:"path"`
	Offset   int64  `json:"offset"`    // byte offset into the file
	MaxBytes int    `json:"max_bytes"` // optional per-call cap (defaults to executor maxRead)
}

type readResult struct {
	Path       string `json:"path"`
	Content    string `json:"content,omitempty"`
	Truncated  bool   `json:"truncated"`
	Truncation string `json:"truncation_reason,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
}

// ReadFile reads text from a repository file with a byte cap.
func (e *Executor) ReadFile(argsJSON []byte) (string, error) {
	var args ReadArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("read_file: bad arguments: %w", err)
	}
	if args.Path == "" {
		return readJSON(readResult{SkipReason: "path is required", Skipped: true}), nil
	}

	abs, rel, err := e.rules.ResolveReadable(args.Path)
	if err != nil {
		return readJSON(readResult{Path: args.Path, SkipReason: err.Error(), Skipped: true}), nil
	}

	f, err := os.Open(abs)
	if err != nil {
		return readJSON(readResult{Path: rel, SkipReason: fmt.Sprintf("open: %v", err), Skipped: true}), nil
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		return readJSON(readResult{Path: rel, SkipReason: fmt.Sprintf("stat: %v", err), Skipped: true}), nil
	}
	if st.Size() > 0 && args.Offset >= st.Size() {
		return readJSON(readResult{Path: rel, Content: "", Truncation: "offset past end of file"}), nil
	}
	if _, err := f.Seek(args.Offset, io.SeekStart); err != nil {
		return readJSON(readResult{Path: rel, SkipReason: fmt.Sprintf("seek: %v", err), Skipped: true}), nil
	}

	max := e.maxRead
	if args.MaxBytes > 0 && args.MaxBytes < max {
		max = args.MaxBytes
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(max)))
	if err != nil {
		return readJSON(readResult{Path: rel, SkipReason: fmt.Sprintf("read: %v", err), Skipped: true}), nil
	}
	n := len(data)

	if bytes.IndexByte(data, 0) >= 0 {
		return readJSON(readResult{
			Path:       rel,
			Skipped:    true,
			SkipReason: "binary file skipped",
		}), nil
	}
	if !utf8.Valid(data) {
		return readJSON(readResult{
			Path:       rel,
			Skipped:    true,
			SkipReason: "non-utf8 file skipped",
		}), nil
	}

	truncated := false
	truncReason := ""
	remain := st.Size() - args.Offset - int64(n)
	if remain > 0 || n >= max {
		truncated = true
		if remain > 0 {
			truncReason = fmt.Sprintf("%d bytes not shown beyond max_read_bytes window", remain)
		} else {
			truncReason = "read window filled max_read_bytes"
		}
	}

	out := readResult{
		Path:       rel,
		Content:    string(data),
		Truncated:  truncated,
		Truncation: truncReason,
	}
	return readJSON(out), nil
}

func readJSON(r readResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(b)
}
