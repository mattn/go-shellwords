package shellwords

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var envRe = regexp.MustCompile(`\$({\w\+}|\w+)`)

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func replaceEnv(s string) string {
	return envRe.ReplaceAllStringFunc(s, func(s string) string {
		s = s[1:]
		if s[0] == '{' {
			s = s[1:len(s)-2]
		}
		return os.Getenv(s)
	})
}

func shellRun(line string) (string, error) {
	if runtime.GOOS == "windows" {
		shell := os.Getenv("COMSPEC")
		b, err := exec.Command(shell, "/c", line).Output()
		if err != nil {
			return "", errors.New(err.Error() + ":" + string(b))
		}
		return string(b), nil
	}
	shell := os.Getenv("SHELL")
	b, err := exec.Command(shell, "-c", line).Output()
	return string(b), errors.New(err.Error() + ":" + string(b))
}

type Parser struct {
	ParseEnv      bool
	ParseBacktick bool
}

func NewParser() *Parser {
	return new(Parser)
}

func (p *Parser) Parse(line string) ([]string, error) {
	line = strings.TrimSpace(line)

	args := []string{}
	buf := ""
	var escaped, doubleQuoted, singleQuoted, backQuote bool
	backtick := ""

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
			if singleQuoted || doubleQuoted || backQuote {
				buf += string(r)
				backtick += string(r)
			} else if buf != "" {
				if p.ParseEnv {
					buf = replaceEnv(buf)
				}
				args = append(args, buf)
				buf = ""
			}
			continue
		}

		switch r {
		case '`':
			if p.ParseBacktick && !singleQuoted && !doubleQuoted {
				if backQuote {
					out, err := shellRun(backtick)
					if err != nil {
						return nil, err
					}
					buf = out
				}
				backtick = ""
				backQuote = !backQuote
				continue
			}
		case '"':
			if !singleQuoted {
				doubleQuoted = !doubleQuoted
				continue
			}
		case '\'':
			if !doubleQuoted {
				singleQuoted = !singleQuoted
				continue
			}
		}

		buf += string(r)
		if backQuote {
			backtick += string(r)
		}
	}

	if buf != "" {
		if p.ParseEnv {
			buf = replaceEnv(buf)
		}
		args = append(args, buf)
	}

	if escaped || singleQuoted || doubleQuoted || backQuote {
		return nil, errors.New("invalid command line string")
	}

	return args, nil
}

func Parse(line string) ([]string, error) {
	return NewParser().Parse(line)
}
