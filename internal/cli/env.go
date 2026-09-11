package cli

import (
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"golang.org/x/term"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/paths"
)

// env is everything a command needs from the outside world.
type env struct {
	Layout   paths.Layout
	Log      *audit.Log
	Home     string
	Engineer string
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	GOOS     string
	Exe      string
	HostEnv  []string
	Workdir  string
	IsTTY    bool
}

func loadEnv() (*env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	l := paths.Resolve(home)
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	name := ""
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	wd, _ := os.Getwd()
	return &env{
		Layout: l, Log: audit.Open(l.Audit, l.AuditHead), Home: home, Engineer: name,
		Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, GOOS: runtime.GOOS, Exe: exe,
		HostEnv: os.Environ(), Workdir: wd, IsTTY: term.IsTerminal(int(os.Stdin.Fd())),
	}, nil
}
