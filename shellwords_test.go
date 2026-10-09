package shellwords

import (
	"errors"
	"go/build"
	"os"
	"os/exec"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type testcase struct {
	line     string
	expected []string
}

var testcases = []testcase{
	{``, []string{}},
	{`""`, []string{``}},
	{`''`, []string{``}},
	{`var --bar=baz`, []string{`var`, `--bar=baz`}},
	{`var --bar="baz"`, []string{`var`, `--bar=baz`}},
	{`var "--bar=baz"`, []string{`var`, `--bar=baz`}},
	{`var "--bar='baz'"`, []string{`var`, `--bar='baz'`}},
	{"var --bar=`baz`", []string{`var`, "--bar=`baz`"}},
	{`var "--bar=\"baz'"`, []string{`var`, `--bar="baz'`}},
	{`var "--bar=\'baz\'"`, []string{`var`, `--bar='baz'`}},
	{`var --bar='\'`, []string{`var`, `--bar=\`}},
	{`var "--bar baz"`, []string{`var`, `--bar baz`}},
	{`var --"bar baz"`, []string{`var`, `--bar baz`}},
	{`var  --"bar baz"`, []string{`var`, `--bar baz`}},
	{`a "b"`, []string{`a`, `b`}},
	{`a " b "`, []string{`a`, ` b `}},
	{`a "   "`, []string{`a`, `   `}},
	{`a 'b'`, []string{`a`, `b`}},
	{`a ' b '`, []string{`a`, ` b `}},
	{`a '   '`, []string{`a`, `   `}},
	{"foo bar\\  ", []string{`foo`, `bar `}},
	{`foo "" bar ''`, []string{`foo`, ``, `bar`, ``}},
	{`foo \\`, []string{`foo`, `\`}},
	{`foo \& bar`, []string{`foo`, `&`, `bar`}},
	{`sh -c "printf 'Hello\tworld\n'"`, []string{`sh`, `-c`, "printf 'Hello\tworld\n'"}},
}

func TestSimple(t *testing.T) {
	for _, testcase := range testcases {
		args, err := Parse(testcase.line)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(args, testcase.expected) {
			t.Fatalf("Expected %#v for %q, but %#v:", testcase.expected, testcase.line, args)
		}
	}
}

func TestComment(t *testing.T) {
	allCases := append(testcases, []testcase{
		{"# comment", []string{}},
		{"foo not#comment", []string{"foo", "not#comment"}},
		{`foo "bar # baz" # comment`, []string{"foo", "bar # baz"}},
		{"foo \"bar # baz\" # comment\nfoo\nbar # baz",
			[]string{"foo", "bar # baz", "foo", "bar"}},
		{"echo '# list all files' # line\\ncomment\n# whole line comment\nls -al '#' # more comment",
			[]string{"echo", "# list all files", "ls", "-al", "#"}},
	}...)

	parser := NewParser()
	parser.ParseComment = true
	for _, testcase := range allCases {
		args, err := parser.Parse(testcase.line)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(args, testcase.expected) {
			t.Fatalf("Expected %#v for %q, but %#v:", testcase.expected, testcase.line, args)
		}
	}
}

func TestCommentInBacktick(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sh")
	}
	parser := NewParser()
	parser.ParseComment = true
	parser.ParseBacktick = true
	args, err := parser.Parse("echo `# x` done")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "done"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEmptySubstitution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sh")
	}
	parser := NewParser()
	parser.ParseBacktick = true
	args, err := parser.Parse("echo `true` done $(true) \"\"")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "done", ""}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestExcludeSeparators(t *testing.T) {
	parser := NewParser()
	parser.SetExcludeSeparators(';', '\t')
	args, err := parser.Parse("a;b c\td e; \"f;g\"")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"a;b", "c\td", "e;", "f;g"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
	if got := string(parser.ExcludedSeparators()); got != ";\t" {
		t.Fatalf("Expected %q, but %q:", ";\t", got)
	}
}

func TestError(t *testing.T) {
	_, err := Parse("foo '")
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = Parse(`foo "`)
	if err == nil {
		t.Fatal("Should be an error")
	}

	_, err = Parse("foo `")
	if err == nil {
		t.Fatal("Should be an error")
	}
}

func TestShellRun(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := shellRun("pwd", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("pwd", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`pwd` should be changed")
	}
}

func TestShellRunNoEnv(t *testing.T) {
	old := os.Getenv("SHELL")
	defer os.Setenv("SHELL", old)
	os.Unsetenv("SHELL")

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := shellRun("pwd", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("pwd", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`pwd` should be changed")
	}
}

func TestBacktick(t *testing.T) {
	goversion, err := shellRun("go version", "")
	if err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	parser.ParseBacktick = true
	args, err := parser.Parse("echo `go version`")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", goversion}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse(`echo $(echo foo)`)
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "foo"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse(`echo bar=$(echo 200)cm`)
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "bar=200cm"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	parser.ParseBacktick = false
	args, err = parser.Parse(`echo $(echo "foo")`)
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", `$(echo "foo")`}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
	args, err = parser.Parse("echo $(`echo1)")
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "$(`echo1)"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestBacktickQuoted(t *testing.T) {
	parser := NewParser()

	args, err := parser.Parse("echo `echo \"foo  bar\"`")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "`echo \"foo  bar\"`"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	if runtime.GOOS == "windows" {
		t.Skip("cmd.exe does not strip quotes")
	}

	parser.ParseBacktick = true
	args, err = parser.Parse("echo `echo \"foo  bar\"`")
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "foo  bar"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse("echo `echo 'foo  bar'`")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestBacktickAfterQuoted(t *testing.T) {
	parser := NewParser()
	parser.ParseBacktick = true

	args, err := parser.Parse(`echo "a b" c$(echo x)d`)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "a b", "cxd"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse(`echo "a b" $(echo x)`)
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "a b", "x"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestSubstitutionEscaped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cmd.exe does not interpret backslash escapes")
	}
	parser := NewParser()
	parser.ParseBacktick = true

	args, err := parser.Parse(`echo $(echo a\ b)`)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "a b"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse("echo `echo a\\  b`")
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "a  b"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestBacktickMulti(t *testing.T) {
	parser := NewParser()
	parser.ParseBacktick = true
	args, err := parser.Parse(`echo $(go env GOPATH && go env GOROOT)`)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", build.Default.GOPATH + "\n" + build.Default.GOROOT}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestBacktickError(t *testing.T) {
	parser := NewParser()
	parser.ParseBacktick = true
	_, err := parser.Parse("echo `go Version`")
	if err == nil {
		t.Fatal("Should be an error")
	}
	var eerr *exec.ExitError
	if !errors.As(err, &eerr) {
		t.Fatal("Should be able to unwrap to *exec.ExitError")
	}
	_, err = parser.Parse(`echo $(echo1)`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo FOO=$(echo1)`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo $(echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo $ (echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo (echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo )echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
}

func TestEnv(t *testing.T) {
	os.Setenv("FOO", "bar")

	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $FOO")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "bar"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestCustomEnv(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.Getenv = func(k string) string { return map[string]string{"FOO": "baz"}[k] }
	args, err := parser.Parse("echo $FOO")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "baz"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvMultibyte(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.Getenv = func(k string) string { return map[string]string{"FOO": "bar"}[k] }
	args, err := parser.Parse("echo あい$FOO ${FOO}う")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "あいbar", "barう"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvBareDollar(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.Getenv = func(k string) string { return map[string]string{"FOO": "bar"}[k] }
	args, err := parser.Parse("echo $@x $FOO")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "$@x", "bar"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvEscapedDollar(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.Getenv = func(k string) string { return map[string]string{"FOO": "bar"}[k] }
	args, err := parser.Parse(`echo \$FOO "\$FOO" $FOO`)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "$FOO", "$FOO", "bar"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvQuoteBoundaries(t *testing.T) {
	tests := []struct {
		line  string
		value string
		want  []string
	}{
		{`'$FOO'`, "bar", []string{"$FOO"}},
		{`'${FOO}'`, "bar", []string{"${FOO}"}},
		{`'a\b'`, "bar", []string{`a\b`}},
		{`$FOO"x"`, "bar", []string{"barx"}},
		{`"$FOO"x`, "bar", []string{"barx"}},
		{`$FOO'x'`, "bar", []string{"barx"}},
		{`'$FOO'"$FOO"`, "bar", []string{"$FOObar"}},
		{`"'$FOO'"`, "bar", []string{"'bar'"}},
		{`$FOO"x"`, "bar baz", []string{"bar", "bazx"}},
		{`"$FOO"x`, "bar baz", []string{"bar bazx"}},
		{`'a b'$FOO`, "bar baz", []string{"a bbar", "baz"}},
		{`"a"$FOO`, " bar ", []string{"a", "bar"}},
		{`$FOO"x"`, " bar ", []string{"bar", "x"}},
		{`"a"$FOO"b"`, " ", []string{"a", "b"}},
		{`"$FOO"x`, `a"b\c`, []string{`a"b\cx`}},
		{`"$FOO"x`, "", []string{"x"}},
		{`$FOO""`, "", []string{""}},
		{`'$FOO' $FOO`, "bar", []string{"$FOO", "bar"}},
		{`a\ b"c"`, "", []string{"a bc"}},
		{`\"x"y"`, "", []string{`"xy`}},
		{`$FOO"x"`, `a"b`, []string{`a"bx`}},
		{`$FOO"x"`, `a'b`, []string{`a'bx`}},
		{`$FOO"x"`, `a\b`, []string{`a\bx`}},
		{`$FOO"x"`, `a;b`, []string{`a;bx`}},
		{`a\ b`, "", []string{"a b"}},
		{`$FOO\ x`, "bar baz", []string{"bar", "baz x"}},
		{`\$FOO"x"`, "bar", []string{"$FOOx"}},
		{`$FOO"x"`, "a\tb\rc\nd", []string{"a", "b", "c", "dx"}},
		{`\\$FOO`, "bar", []string{`\bar`}},
		{`"a\\$FOO"`, "bar", []string{`a\bar`}},
		{`""$FOO`, " bar", []string{"", "bar"}},
		{`$FOO$FOO`, "a b", []string{"a", "ba", "b"}},
		{`${FOO}x${FOO}`, " ", []string{"x"}},
		{`"${FOO}"x`, "a b", []string{"a bx"}},
		{`${FOO`, "bar", []string{"${FOO"}},
		{`"${FOO"x`, "bar", []string{"${FOOx"}},
		{`${FOO-x}`, "bar", []string{"${FOO-x}"}},
		{`'$FOO'x`, "bar", []string{"$FOOx"}},
		{`x'$FOO'`, "bar", []string{"x$FOO"}},
		{`'${FOO}'"${FOO}"`, "bar", []string{"${FOO}bar"}},
		{`"$FOO"'$FOO'`, "bar", []string{"bar$FOO"}},
		{`"a'$FOO'b"`, "bar", []string{"a'bar'b"}},
		{`'"$FOO"'`, "bar", []string{`"$FOO"`}},
		{`${FOO}"x"`, "bar", []string{"barx"}},
		{`"x"${FOO}`, "bar", []string{"xbar"}},
		{`あ$FOO"い"`, "bar", []string{"あbarい"}},
		{`$FOO'x'$FOO`, "a b", []string{"a", "bxa", "b"}},
		{`"$FOO""$FOO"`, "a b", []string{"a ba b"}},
		{`a\ b'c'`, "", []string{"a bc"}},
		{`\'x'y'`, "", []string{"'xy"}},
		{`a\\'b'`, "", []string{`a\b`}},
		{`\$FOO'x'`, "bar", []string{"$FOOx"}},
		{`\${FOO}`, "bar", []string{"${FOO}"}},
		{`"\${FOO}"`, "bar", []string{"${FOO}"}},
		{`\\\$FOO`, "bar", []string{`\$FOO`}},
		{`$FOO'x'`, `a"b`, []string{`a"bx`}},
		{`"$FOO"`, `a'b"c\d$e`, []string{`a'b"c\d$e`}},
		{`$FOO`, `\$X`, []string{`\$X`}},
		{`"$FOO"`, "$FOO", []string{"$FOO"}},
		{`$FOO`, "${FOO}", []string{"${FOO}"}},
		{`$FOO x`, "a;b", []string{"a;b", "x"}},
		{`$FOO x`, "a|b&c<d>e", []string{"a|b&c<d>e", "x"}},
		{`$FOO`, `a\tb`, []string{`a\tb`}},
	}
	for _, tt := range tests {
		t.Run(tt.line+"/"+tt.value, func(t *testing.T) {
			parser := NewParser()
			parser.ParseEnv = true
			parser.Getenv = func(key string) string {
				if key == "FOO" {
					return tt.value
				}
				return ""
			}
			args, err := parser.Parse(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, tt.want) {
				t.Fatalf("Expected %#v, but %#v", tt.want, args)
			}
		})
	}
}

func TestEnvSplitDoesNotStartComment(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.ParseComment = true
	parser.Getenv = func(key string) string { return "a " }
	tests := []struct {
		line  string
		value string
		want  []string
	}{
		{"$FOO#x y # z", "a ", []string{"a", "#x", "y"}},
		{"${FOO}#x", "a ", []string{"a", "#x"}},
		{"$FOO#x", "", []string{"#x"}},
		{`"$FOO"#x`, "a ", []string{"a #x"}},
		{"$FOO #x", "a ", []string{"a"}},
		{"$FOO #x", "", []string{}},
		{"x $FOO\n# y\nz", "a", []string{"x", "a", "z"}},
	}
	for _, tt := range tests {
		parser.Getenv = func(string) string { return tt.value }
		args, err := parser.Parse(tt.line)
		if err != nil {
			t.Fatalf("Parse(%q) with FOO=%q: %v", tt.line, tt.value, err)
		}
		if !reflect.DeepEqual(args, tt.want) {
			t.Fatalf("Parse(%q) with FOO=%q: expected %#v, but %#v", tt.line, tt.value, tt.want, args)
		}
	}
}

func TestEnvSeparatorPosition(t *testing.T) {
	tests := []struct {
		line  string
		value string
		want  []string
		pos   int
	}{
		{"$FOO; b", "a b", []string{"a", "b"}, 4},
		{`$FOO"x"; b`, "a b", []string{"a", "bx"}, 7},
		{"${FOO}x| b", "あ い", []string{"あ", "いx"}, 7},
		{"$FOO b", "a;c", []string{"a;c", "b"}, -1},
		{"$FOO>f", "2", []string{"2"}, 4},
		{"${FOO}2>f", "a ", []string{"a", "2"}, 7},
		{"$FOO 2>f", "a", []string{"a"}, 5},
	}
	for _, tt := range tests {
		parser := NewParser()
		parser.ParseEnv = true
		parser.Getenv = func(string) string { return tt.value }
		args, err := parser.Parse(tt.line)
		if err != nil {
			t.Fatalf("Parse(%q) with FOO=%q: %v", tt.line, tt.value, err)
		}
		if !reflect.DeepEqual(args, tt.want) {
			t.Fatalf("Parse(%q) with FOO=%q: expected %#v, but %#v", tt.line, tt.value, tt.want, args)
		}
		if parser.Position != tt.pos {
			t.Fatalf("Parse(%q) with FOO=%q: expected position %d, but %d", tt.line, tt.value, tt.pos, parser.Position)
		}
	}
}

func TestQuotedDollarDoesNotStartSubstitution(t *testing.T) {
	for _, env := range []bool{false, true} {
		parser := NewParser()
		parser.ParseEnv = env
		parser.ParseBacktick = true
		for _, line := range []string{
			`\$(echo x)`, `'$'(echo x)`, `"$"(echo x)`,
			`a\$(echo x)`, `a'$'(echo x)`, `"a$"(echo x)`, `\\\$(echo x)`, `$ (echo x)`,
		} {
			args, err := parser.Parse(line)
			if err == nil {
				t.Fatalf("Parse(%q) with ParseEnv=%v: expected an error, but %#v", line, env, args)
			}
		}
	}
}

func TestEnvValueNotSubstituted(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.ParseBacktick = true
	for _, value := range []string{"$(echo x)", "`echo x`", "a$(echo x)b"} {
		for _, line := range []string{`$FOO`, `"$FOO"`, `$FOO""`} {
			parser.Getenv = func(string) string { return value }
			args, err := parser.Parse(line)
			if err != nil {
				t.Fatalf("Parse(%q) with FOO=%q: %v", line, value, err)
			}
			var want []string
			if line == `"$FOO"` {
				want = []string{value}
			} else {
				want = strings.Fields(value)
			}
			if !reflect.DeepEqual(args, want) {
				t.Fatalf("Parse(%q) with FOO=%q: expected %#v, but %#v", line, value, want, args)
			}
		}
	}
}

func TestEnvQuoteBoundariesExcludedSeparators(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.SetExcludeSeparators(';', '\t')
	parser.Getenv = func(key string) string { return "a\tb;c" }
	for _, line := range []string{"a\tb;c\"x\"", `$FOO"x"`} {
		args, err := parser.Parse(line)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"a\tb;cx"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("Parse(%q): expected %#v, but %#v", line, want, args)
		}
	}

	parser.Getenv = func(key string) string { return "a\tb c\nd" }
	for _, line := range []string{`$FOO`, `${FOO}`, `$FOO""`} {
		args, err := parser.Parse(line)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"a\tb", "c", "d"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("Parse(%q): expected %#v, but %#v", line, want, args)
		}
	}
}

func TestNoEnv(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $BAR")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvArguments(t *testing.T) {
	os.Setenv("FOO", "bar baz")

	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $FOO")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "bar", "baz"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvArgumentsNotReparsed(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.ParseBacktick = true

	tests := []struct {
		line  string
		value string
		want  []string
	}{
		{"echo $FOO", "bar '", []string{"echo", "bar", "'"}},
		{"$FOO ", "bar `", []string{"bar", "`"}},
		{"$FOO", `"a b"`, []string{`"a`, `b"`}},
		{"$FOO", "$(echo PWNED)", []string{"$(echo", "PWNED)"}},
		{"$FOO", "a;b|c>d", []string{"a;b|c>d"}},
	}
	for _, tt := range tests {
		parser.Getenv = func(string) string { return tt.value }
		args, err := parser.Parse(tt.line)
		if err != nil {
			t.Fatalf("Parse(%q) with FOO=%q: %v", tt.line, tt.value, err)
		}
		if !reflect.DeepEqual(args, tt.want) {
			t.Fatalf("Parse(%q) with FOO=%q: expected %#v, but %#v", tt.line, tt.value, tt.want, args)
		}
	}
}

func TestDupEnv(t *testing.T) {
	os.Setenv("FOO", "bar")
	os.Setenv("FOO_BAR", "baz")

	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $FOO$")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "bar$"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse("echo ${FOO_BAR}$")
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "baz$"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestHaveMore(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true

	line := "echo 🍺; seq 1 10"
	args, err := parser.Parse(line)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := []string{"echo", "🍺"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	if parser.Position == 0 {
		t.Fatalf("Commands should be remaining")
	}

	line = string([]rune(line)[parser.Position+1:])
	args, err = parser.Parse(line)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected = []string{"seq", "1", "10"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	if parser.Position > 0 {
		t.Fatalf("Commands should not be remaining")
	}
}

func TestHaveRedirect(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true

	line := "ls -la 2>foo"
	args, err := parser.Parse(line)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := []string{"ls", "-la"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	if parser.Position == 0 {
		t.Fatalf("Commands should be remaining")
	}
}

func TestHaveRedirectPrefix(t *testing.T) {
	parser := NewParser()

	line := "cmd 2x> file"
	args, err := parser.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"cmd", "2x"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
	if rest := line[parser.Position:]; rest != "> file" {
		t.Fatalf("Expected %q, but %q:", "> file", rest)
	}

	line = "cmd 10> file"
	args, err = parser.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"cmd"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
	if rest := line[parser.Position:]; rest != "10> file" {
		t.Fatalf("Expected %q, but %q:", "10> file", rest)
	}
}

func TestHaveInputRedirectPrefix(t *testing.T) {
	tests := []struct {
		line     string
		wantArgs []string
		wantRest string
	}{
		{`cmd 0<file`, []string{"cmd"}, "0<file"},
		{`cmd 10<file`, []string{"cmd"}, "10<file"},
		{`cmd 10<<EOF`, []string{"cmd"}, "10<<EOF"},
		{`cmd 10<<-EOF`, []string{"cmd"}, "10<<-EOF"},
		{`cmd 10<&0`, []string{"cmd"}, "10<&0"},
		{`cmd 10<>file`, []string{"cmd"}, "10<>file"},
		{`cmd 2x<file`, []string{"cmd", "2x"}, "<file"},
		{`cmd 10 <file`, []string{"cmd", "10"}, "<file"},
		{`cmd "10"<file`, []string{"cmd", "10"}, "<file"},
		{`cmd '10'<file`, []string{"cmd", "10"}, "<file"},
		{`cmd 1\0<file`, []string{"cmd", "10"}, "<file"},
		{`cmd 2""<file`, []string{"cmd", "2"}, "<file"},
		{`cmd $FD<file`, []string{"cmd", "10"}, "<file"},
		{`cmd $FD 2<file`, []string{"cmd", "10"}, "2<file"},
		{`cmd "x" 2<file`, []string{"cmd", "x"}, "2<file"},
		{`cmd 🍺 10<file`, []string{"cmd", "🍺"}, "10<file"},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			parser := NewParser()
			parser.ParseEnv = true
			parser.Getenv = func(string) string { return "10" }
			args, err := parser.Parse(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("Expected %#v, but %#v", tt.wantArgs, args)
			}
			if parser.Position < 0 {
				t.Fatalf("Expected a redirect position, but %d", parser.Position)
			}
			if rest := string([]rune(tt.line)[parser.Position:]); rest != tt.wantRest {
				t.Errorf("Expected %q, but %q", tt.wantRest, rest)
			}
		})
	}
}

func TestHaveQuotedRedirectPrefix(t *testing.T) {
	tests := []struct {
		line         string
		parseComment bool
		wantArgs     []string
		wantRest     string
	}{
		{
			line:     `cmd "10">file`,
			wantArgs: []string{"cmd", "10"},
			wantRest: ">file",
		},
		{
			line:     `cmd '10'>file`,
			wantArgs: []string{"cmd", "10"},
			wantRest: ">file",
		},
		{
			line:     `cmd 1\0>file`,
			wantArgs: []string{"cmd", "10"},
			wantRest: ">file",
		},
		{
			line:     `cmd 2"">file`,
			wantArgs: []string{"cmd", "2"},
			wantRest: ">file",
		},
		{
			line:     `cmd ""2>file`,
			wantArgs: []string{"cmd", "2"},
			wantRest: ">file",
		},
		{
			// Quoting earlier in the line must not disarm a later descriptor.
			line:     `cmd "x" 2>file`,
			wantArgs: []string{"cmd", "x"},
			wantRest: "2>file",
		},
		{
			// Since #77 a '#' after empty quotes is word content, not a comment
			// start, so this yields the "#comment" argument. It still exercises
			// the token boundary: tokenQuoted must be cleared before the 2> on
			// the next line, or that descriptor would be misread as quoted.
			line:         "cmd \"\"#comment\n2>file",
			parseComment: true,
			wantArgs:     []string{"cmd", "#comment"},
			wantRest:     "2>file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			parser := NewParser()
			parser.ParseComment = tt.parseComment
			args, err := parser.Parse(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("Expected %#v, but %#v", tt.wantArgs, args)
			}
			if rest := tt.line[parser.Position:]; rest != tt.wantRest {
				t.Errorf("Expected %q, but %q", tt.wantRest, rest)
			}
		})
	}
}

func TestHaveSubstitutedRedirectPrefix(t *testing.T) {
	tests := []struct {
		line     string
		wantArgs []string
		wantRest string
	}{
		{
			line:     "cmd $(printf 2)>file",
			wantArgs: []string{"cmd", "2"},
			wantRest: ">file",
		},
		{
			line:     "cmd `printf 2`>file",
			wantArgs: []string{"cmd", "2"},
			wantRest: ">file",
		},
		{
			// Output longer than the line must not rewind Position below zero.
			line:     "cmd $(printf %040d 0)>file",
			wantArgs: []string{"cmd", strings.Repeat("0", 40)},
			wantRest: ">file",
		},
		{
			// A substitution earlier in the line must not disarm a later descriptor.
			line:     "cmd $(printf x) 2>file",
			wantArgs: []string{"cmd", "x"},
			wantRest: "2>file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			parser := NewParser()
			parser.ParseBacktick = true
			args, err := parser.Parse(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("Expected %#v, but %#v", tt.wantArgs, args)
			}
			if parser.Position < 0 || parser.Position > len(tt.line) {
				t.Fatalf("Position out of range: %d", parser.Position)
			}
			if rest := tt.line[parser.Position:]; rest != tt.wantRest {
				t.Errorf("Expected %q, but %q", tt.wantRest, rest)
			}
		})
	}
}

func TestBackquoteInFlag(t *testing.T) {
	parser := NewParser()
	parser.ParseBacktick = true
	args, err := parser.Parse("cmd -flag=`echo val1` -flag=val2")
	if err != nil {
		panic(err)
	}
	expected := []string{"cmd", "-flag=val1", "-flag=val2"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvInQuoted(t *testing.T) {
	os.Setenv("FOO", "bar")

	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse(`ssh 127.0.0.1 "echo $FOO"`)
	if err != nil {
		panic(err)
	}
	expected := []string{"ssh", "127.0.0.1", "echo bar"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse(`ssh 127.0.0.1 "echo \$FOO"`)
	if err != nil {
		panic(err)
	}
	expected = []string{"ssh", "127.0.0.1", "echo $FOO"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse(`ssh 127.0.0.1 "echo \\$FOO"`)
	if err != nil {
		panic(err)
	}
	expected = []string{"ssh", "127.0.0.1", `echo \bar`}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestParseWithEnvs(t *testing.T) {
	tests := []struct {
		line               string
		wantEnvs, wantArgs []string
	}{
		{
			line:     "FOO=foo cmd --args=A=B",
			wantEnvs: []string{"FOO=foo"},
			wantArgs: []string{"cmd", "--args=A=B"},
		},
		{
			line:     "FOO=foo BAR=bar cmd --args=A=B -A=B",
			wantEnvs: []string{"FOO=foo", "BAR=bar"},
			wantArgs: []string{"cmd", "--args=A=B", "-A=B"},
		},
		{
			line:     `sh -c "FOO=foo BAR=bar cmd --args=A=B -A=B"`,
			wantEnvs: []string{},
			wantArgs: []string{"sh", "-c", "FOO=foo BAR=bar cmd --args=A=B -A=B"},
		},
		{
			line:     "cmd --args=A=B -A=B",
			wantEnvs: []string{},
			wantArgs: []string{"cmd", "--args=A=B", "-A=B"},
		},
		{
			line:     "FOO=a=b cmd",
			wantEnvs: []string{"FOO=a=b"},
			wantArgs: []string{"cmd"},
		},
		{
			line:     "LS_COLORS=di=34:ln=35 ls",
			wantEnvs: []string{"LS_COLORS=di=34:ln=35"},
			wantArgs: []string{"ls"},
		},
		{
			line:     "=x 1a=b cmd",
			wantEnvs: []string{},
			wantArgs: []string{"=x", "1a=b", "cmd"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			envs, args, err := ParseWithEnvs(tt.line)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(envs, tt.wantEnvs) {
				t.Errorf("Expected %#v, but %#v", tt.wantEnvs, envs)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("Expected %#v, but %#v", tt.wantArgs, args)
			}
		})
	}
}

func TestSubShellEnv(t *testing.T) {
	myParser := &Parser{
		ParseEnv: true,
	}

	errTmpl := "bad arg parsing:\nexpected: %#v\nactual  : %#v\n"

	t.Run("baseline", func(t *testing.T) {
		args, err := myParser.Parse(`program -f abc.txt`)
		if err != nil {
			t.Fatalf("err should be nil: %v", err)
		}
		expected := []string{"program", "-f", "abc.txt"}
		if len(args) != 3 {
			t.Fatalf(errTmpl, expected, args)
		}
		if args[0] != expected[0] || args[1] != expected[1] || args[2] != expected[2] {
			t.Fatalf(errTmpl, expected, args)
		}
	})

	t.Run("single-quoted", func(t *testing.T) {
		args, err := myParser.Parse(`sh -c 'echo foo'`)
		if err != nil {
			t.Fatalf("err should be nil: %v", err)
		}
		expected := []string{"sh", "-c", "echo foo"}
		if len(args) != 3 {
			t.Fatalf(errTmpl, expected, args)
		}
		if args[0] != expected[0] || args[1] != expected[1] || args[2] != expected[2] {
			t.Fatalf(errTmpl, expected, args)
		}
	})

	t.Run("double-quoted", func(t *testing.T) {
		args, err := myParser.Parse(`sh -c "echo foo"`)
		if err != nil {
			t.Fatalf("err should be nil: %v", err)
		}
		expected := []string{"sh", "-c", "echo foo"}
		if len(args) != 3 {
			t.Fatalf(errTmpl, expected, args)
		}
		if args[0] != expected[0] || args[1] != expected[1] || args[2] != expected[2] {
			t.Fatalf(errTmpl, expected, args)
		}
	})
}

func TestCommentAfterEmptyQuotedWord(t *testing.T) {
	parser := NewParser()
	parser.ParseComment = true
	for _, tc := range []struct {
		line string
		want []string
	}{
		{`echo ''#literal`, []string{"echo", "#literal"}},
		{`echo ""#literal`, []string{"echo", "#literal"}},
		{`echo ''#literal # comment`, []string{"echo", "#literal"}},
		{`echo '' # comment`, []string{"echo", ""}},
		{`echo # comment`, []string{"echo"}},
	} {
		got, err := parser.Parse(tc.line)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Parse(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}
