# go-shellwords

[![codecov](https://codecov.io/gh/mattn/go-shellwords/branch/master/graph/badge.svg)](https://codecov.io/gh/mattn/go-shellwords)
[![Build Status](https://travis-ci.org/mattn/go-shellwords.svg?branch=master)](https://travis-ci.org/mattn/go-shellwords)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/mattn/go-shellwords)](https://pkg.go.dev/github.com/mattn/go-shellwords)
[![ci](https://github.com/mattn/go-shellwords/ci/badge.svg)](https://github.com/mattn/go-shellwords/actions)

Parse line as shell words.

## Usage

```go
args, err := shellwords.Parse("./foo --bar=baz")
// args should be ["./foo", "--bar=baz"]
```

```go
envs, args, err := shellwords.ParseWithEnvs("FOO=foo BAR=baz ./foo --bar=baz")
// envs should be ["FOO=foo", "BAR=baz"]
// args should be ["./foo", "--bar=baz"]
```

```go
os.Setenv("FOO", "bar")
p := shellwords.NewParser()
p.ParseEnv = true
args, err := p.Parse("./foo $FOO")
// args should be ["./foo", "bar"]
```

```go
p := shellwords.NewParser()
p.ParseBacktick = true
args, err := p.Parse("./foo `echo $SHELL`")
// args should be ["./foo", "/bin/bash"]
```

```go
shellwords.ParseBacktick = true
p := shellwords.NewParser()
args, err := p.Parse("./foo `echo $SHELL`")
// args should be ["./foo", "/bin/bash"]
```

```go
p := shellwords.NewParser()
p.ParseComment = true
args, err := p.Parse("./foo # comment")
// args should be ["./foo"]
```

```go
p := shellwords.NewParser()
p.SetExcludeSeparators('\t', ';')
```

To return the complete sequence of words and operators, opt in on a parser:

```go
p := shellwords.NewParser()
p.ParseOperators = true
args, err := p.Parse("cmd>>out||next 2>&1")
// args should be ["cmd", ">>", "out", "||", "next", "2", ">&", "1"]
// p.Position is -1 after a successful complete parse.
```

`ParseOperators` defaults to false. Without it, parsing still stops at the
first unquoted operator and `Position` identifies the remaining input.
With it, operators use the longest match from `<<-`, `&&`, `||`, `;;`, `;&`,
`<<`, `>>`, `<&`, `>&`, `<>`, `>|`, `;`, `&`, `|`, `<`, and `>`.
Other combinations split into these tokens: `<<<` becomes `<<`, `<`,
`&>` becomes `&`, `>`, and `;;&` becomes `;;`, `&`. Numeric words remain
separate tokens, as in `2>&1` above. Quoted or escaped punctuation stays
inside words. `SetExcludeSeparators` takes priority, including within a
multi-character operator: excluding `>` makes `cmd>out|next` return
`["cmd>out", "|", "next"]`.

`ParseWithEnvs` still returns a single leading environment list. An operator
ends that prefix; it does not restart after a pipe or treat redirect targets
specially. For example, `A=1 cmd|B=2 next` gives envs `["A=1"]` and args
`["cmd", "|", "B=2", "next"]`, while `>out A=1 cmd` gives no envs and args
`[">", "out", "A=1", "cmd"]`. With `ParseEnv` enabled and a `Getenv` callback
that returns `"a b"` for `VALUE`, `A=1 cmd|B=$VALUE next` gives envs `["A=1"]`
and args `["cmd", "|", "B=a", "b", "next"]`.

This option tokenizes operators; it does not validate shell syntax, execute
pipelines or redirections, process here-documents, or add parenthesis syntax.
Existing whitespace, comments when enabled, quotes, escapes and errors retain
their behavior. It does not enable `ParseEnv` or `ParseBacktick`; leave
`ParseBacktick` disabled to avoid command execution. Disabled backtick and
`$(...)` regions remain literal, and expansion output is never reinterpreted
as operator syntax.

# Thanks

This is based on cpan module [Parse::CommandLine](https://metacpan.org/pod/Parse::CommandLine).

# License

under the MIT License: http://mattn.mit-license.org/2017

# Author

Yasuhiro Matsumoto (a.k.a mattn)
