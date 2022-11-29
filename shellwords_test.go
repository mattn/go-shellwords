package shellwords

import (
	"errors"
	"go/build"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
)

var testcases = []struct {
	line     string
	expected []string
}{
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

func TestEnvArgumentsFail(t *testing.T) {
	os.Setenv("FOO", "bar '")

	parser := NewParser()
	parser.ParseEnv = true
	_, err := parser.Parse("echo $FOO")
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse("$FOO")
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse("echo $FOO")
	if err == nil {
		t.Fatal("Should be an error")
	}
	os.Setenv("FOO", "bar `")
	result, err := parser.Parse("$FOO ")
	if err == nil {
		t.Fatal("Should be an error: ", result)
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

	args, err = parser.Parse(`ssh 127.0.0.1 "echo \\$FOO"`)
	if err != nil {
		panic(err)
	}
	expected = []string{"ssh", "127.0.0.1", "echo $FOO"}
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
