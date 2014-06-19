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

	var escaped, double_quoted, single_quoted bool

	for _, r := range line {
		if escaped {
			buf += string(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if single_quoted {
				buf += string(r)
			} else {
				escaped = true
			}
			continue
		}

		if isSpace(r) {
			if single_quoted || double_quoted {
				buf += string(r)
			} else if buf != "" {
				args = append(args, buf)
				buf = ""
			}
			continue
		}

		if r == '"' {
			if single_quoted {
				buf += string(r)
			} else {
				double_quoted = !double_quoted
			}
			continue
		}

		if r == '\'' {
			if double_quoted {
				buf += string(r)
				continue
			} else {
				single_quoted = !single_quoted
			}
			continue
		}

		buf += string(r)
	}

	if buf != "" {
		args = append(args, buf)
	}

	if escaped || single_quoted || double_quoted {
		return nil, errors.New("invalid command line string")
	}

	return args, nil
}
