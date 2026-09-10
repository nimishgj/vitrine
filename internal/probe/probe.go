// Package probe runs inside a sandbox and reports what it can and cannot do.
package probe

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Params are the paths and addresses to test against.
type Params struct {
	CredPath       string // engineer credential file that must be unreadable
	HomeDir        string // engineer home that must not list
	StateFile      string // ~/.vitrine/grants.json (or audit.jsonl) that must be unreadable
	OutsidePath    string // a file path outside every grant that must not be creatable
	WriteGrant     string // directory with write grant
	ReadGrant      string // directory with read grant containing a file "readme"
	AgentHome      string
	Scratch        string
	SymlinkInGrant string // symlink inside WriteGrant whose target is outside
	CanaryEnv      string // env var name that the host exported and must be absent
	NetAddr        string // host:port that must be reachable
}

// Check is one probe result.
type Check struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Want   string `json:"want"`
	Got    string `json:"got"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

func result(id int, name, want string, err error, allowedWhenNil bool) Check {
	got := "denied"
	if err == nil {
		got = "allowed"
	}
	if !allowedWhenNil { // for "absent"-style checks err==nil means present
		if err == nil {
			got = "present"
		} else {
			got = "absent"
		}
	}
	c := Check{ID: id, Name: name, Want: want, Got: got, OK: got == want}
	if err != nil {
		c.Detail = err.Error()
	}
	return c
}

func tryRead(p string) error {
	_, err := os.ReadFile(p)
	return err
}

func tryWrite(p string) error {
	if err := os.WriteFile(p, []byte("vitrine-probe"), 0o600); err != nil {
		return err
	}
	os.Remove(p)
	return nil
}

func tryList(dir string) error {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(ents) == 0 {
		return errors.New("empty")
	}
	return nil
}

// Run executes every check and returns the results in a fixed order.
func Run(p Params) []Check {
	var cs []Check
	cs = append(cs, result(1, "read engineer credential", "denied", tryRead(p.CredPath), true))
	cs = append(cs, result(2, "list engineer home", "denied", tryList(p.HomeDir), true))
	cs = append(cs, result(3, "read vitrine state file", "denied", tryRead(p.StateFile), true))
	cs = append(cs, result(4, "write outside any grant", "denied", tryWrite(p.OutsidePath), true))
	cs = append(cs, result(5, "write inside write grant", "allowed", tryWrite(filepath.Join(p.WriteGrant, ".vitrine-probe")), true))
	cs = append(cs, result(6, "write inside read grant", "denied", tryWrite(filepath.Join(p.ReadGrant, ".vitrine-probe")), true))
	cs = append(cs, result(7, "read inside read grant", "allowed", tryRead(filepath.Join(p.ReadGrant, "readme")), true))
	cs = append(cs, result(8, "write .git/hooks in write grant", "denied", tryWrite(filepath.Join(p.WriteGrant, ".git", "hooks", "vitrine-probe")), true))
	cs = append(cs, result(9, "write .git/config in write grant", "denied", appendTo(filepath.Join(p.WriteGrant, ".git", "config")), true))
	cs = append(cs, result(10, "follow symlink out of write grant", "denied", tryRead(p.SymlinkInGrant), true))
	cs = append(cs, result(11, "write agent state home", "allowed", tryWrite(filepath.Join(p.AgentHome, ".vitrine-probe")), true))
	cs = append(cs, result(12, "write scratch", "allowed", tryWrite(filepath.Join(p.Scratch, "vitrine-probe")), true))
	if runtime.GOOS == "linux" {
		cs = append(cs, result(13, "TIOCSTI keystroke injection", "denied", tryTIOCSTI(), true))
	}
	var envErr error
	if _, ok := os.LookupEnv(p.CanaryEnv); !ok {
		envErr = errors.New("not set")
	}
	cs = append(cs, result(14, "host environment canary", "absent", envErr, false))
	cs = append(cs, result(15, "outbound tcp", "allowed", tryDial(p.NetAddr), true))
	return cs
}

func appendTo(p string) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

func tryDial(addr string) error {
	if addr == "" {
		return errors.New("no address")
	}
	c, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return err
	}
	return c.Close()
}

// Passed is true when every check matched its expectation.
func Passed(cs []Check) bool {
	for _, c := range cs {
		if !c.OK {
			return false
		}
	}
	return true
}

// Format renders checks for terminal output.
func Format(cs []Check) string {
	s := ""
	for _, c := range cs {
		mark := "ok  "
		if !c.OK {
			mark = "FAIL"
		}
		s += fmt.Sprintf("%s  %2d  %-36s want %-7s got %s\n", mark, c.ID, c.Name, c.Want, c.Got)
	}
	return s
}
