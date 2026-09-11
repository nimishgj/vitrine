package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/grants"
)

func TestPromptTrust(t *testing.T) {
	cases := []struct {
		in      string
		mode    grants.Mode
		proceed bool
		err     bool
	}{
		{"w\n", grants.ModeWrite, true, false},
		{"r\n", grants.ModeRead, true, false},
		{"n\n", "", false, false},
		{"x\nw\n", grants.ModeWrite, true, false}, // re-asks on junk
		{"", "", false, true},                     // EOF
	}
	for _, c := range cases {
		out := &bytes.Buffer{}
		mode, proceed, err := promptTrust(strings.NewReader(c.in), out, true, "claude", "/x/repo")
		if (err != nil) != c.err || mode != c.mode || proceed != c.proceed {
			t.Errorf("%q: mode=%q proceed=%v err=%v", c.in, mode, proceed, err)
		}
		if !strings.Contains(out.String(), "/x/repo") {
			t.Errorf("prompt must name the directory")
		}
	}
	if _, _, err := promptTrust(strings.NewReader("w\n"), &bytes.Buffer{}, false, "claude", "/x"); err == nil {
		t.Fatal("non-tty must fail closed")
	}
}
