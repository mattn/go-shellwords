package shellwords

import (
	"errors"
	"strings"
)

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func Parse(line string) ([]string, error) {
	line = strings.TrimSpace(line)

	args := []string{}

	buf := ""

	var escaped, doubleQuoted, singleQuoted bool

	for _, r := range line {
		if escaped {
			buf += string(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if singleQuoted {
				buf += string(r)
			} else {
				escaped = true
			}
			continue
		}

		if isSpace(r) {
			if singleQuoted || doubleQuoted {
				buf += string(r)
			} else if buf != "" {
				args = append(args, buf)
				buf = ""
			}
			continue
		}

		if r == '"' {
			if singleQuoted {
				buf += string(r)
			} else {
				doubleQuoted = !doubleQuoted
			}
			continue
		}

		if r == '\'' {
			if doubleQuoted {
				buf += string(r)
				continue
			} else {
				singleQuoted = !singleQuoted
			}
			continue
		}

		buf += string(r)
	}

	if buf != "" {
		args = append(args, buf)
	}

	if escaped || singleQuoted || doubleQuoted {
		return nil, errors.New("invalid command line string")
	}

	return args, nil
}
