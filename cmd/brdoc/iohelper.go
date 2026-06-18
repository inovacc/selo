package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxLine = 1024 * 1024

// scanBuf is a scratch buffer reused by bufio.Scanner for long lines.
var scanBuf = make([]byte, 0, 64*1024)

// lineValidator validates a single trimmed line. It returns the formatted
// representation (empty string when no mask applies) and whether it is valid.
type lineValidator func(value string) (formatted string, valid bool)

// openReader returns an io.Reader for the given path. If path is "-" it returns
// stdin and a nil close function; otherwise it opens the absolute path and
// returns a close function that ignores the close error.
func openReader(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, nil, nil
	}

	fullPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { _ = f.Close() }, nil
}

// streamValidate scans r line by line (max 1 MB per line), trims whitespace,
// skips blank lines and '#'-prefixed comments, and writes one result line per
// input: "valid\t<formatted>" (or bare "valid" when formatted is empty) for
// valid values and "invalid\t<value>" otherwise. It returns whether any line
// was invalid and any scanner error.
func streamValidate(r io.Reader, w io.Writer, fn lineValidator) (bool, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(scanBuf, maxLine)

	bw := bufio.NewWriter(w)
	defer func() { _ = bw.Flush() }()

	anyInvalid := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		formatted, valid := fn(line)
		switch {
		case valid && formatted != "":
			_, _ = fmt.Fprintf(bw, "valid\t%s\n", formatted)
		case valid:
			_, _ = fmt.Fprintln(bw, "valid")
		default:
			anyInvalid = true
			_, _ = fmt.Fprintf(bw, "invalid\t%s\n", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return anyInvalid, err
	}

	return anyInvalid, nil
}
