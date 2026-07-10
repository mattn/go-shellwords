package shellwords

import "testing"

func benchmarkParse(b *testing.B, parser *Parser, line string) {
	b.Helper()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := parser.Parse(line); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSimple(b *testing.B) {
	benchmarkParse(b, NewParser(), `go test ./... -run TestSimple -count=1`)
}

func BenchmarkParseQuoted(b *testing.B) {
	benchmarkParse(b, NewParser(), `sh -c "printf 'Hello\tworld\n' && echo done"`)
}

func BenchmarkParseUnicode(b *testing.B) {
	benchmarkParse(b, NewParser(), `echo "🍺 café 東京" --name='mattn'`)
}

func BenchmarkParseEnv(b *testing.B) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.Getenv = func(k string) string {
		switch k {
		case "HOME":
			return "/tmp/home"
		case "GOOS":
			return "linux"
		case "GOARCH":
			return "amd64"
		default:
			return ""
		}
	}
	benchmarkParse(b, parser, `env HOME=$HOME GOOS=$GOOS GOARCH=$GOARCH sh -c "echo $HOME/$GOOS/$GOARCH"`)
}

func BenchmarkParseWithEnvs(b *testing.B) {
	b.ReportAllocs()
	line := `FOO=foo BAR=bar ./cmd --flag=value "quoted arg"`
	for i := 0; i < b.N; i++ {
		if _, _, err := ParseWithEnvs(line); err != nil {
			b.Fatal(err)
		}
	}
}
