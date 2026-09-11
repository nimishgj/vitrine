package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/config"
	"github.com/nimishgj/vitrine/internal/provider"
	"github.com/nimishgj/vitrine/internal/provider/fake"
)

func TestTargetAddListRemove(t *testing.T) {
	e := testEnv(t)
	newInitCmd(e).Execute()
	provider.Register(&fake.Provider{})

	c := newTargetCmd(e)
	c.SetArgs([]string{"add", "prod", "--kind", "fake", "--set", "context=prod-eu", "--set", "ns=default"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load(e.Layout.Config)
	if len(cfg.Targets) != 1 || cfg.Targets[0].Settings["context"] != "prod-eu" || cfg.Targets[0].Settings["ns"] != "default" {
		t.Fatalf("%+v", cfg.Targets)
	}
	evs, _ := e.Log.ReadAll()
	if evs[len(evs)-1].Type != audit.TypeConfigWrite {
		t.Fatal("config.write not logged")
	}

	c = newTargetCmd(e)
	c.SetArgs([]string{"add", "x", "--kind", "nope"})
	if err := c.Execute(); err == nil {
		t.Fatal("unknown kind must fail")
	}

	out := &bytes.Buffer{}
	e.Stdout = out
	c = newTargetCmd(e)
	c.SetArgs([]string{"list"})
	c.Execute()
	if !strings.Contains(out.String(), "prod") || !strings.Contains(out.String(), "fake") {
		t.Fatalf("list: %s", out.String())
	}

	c = newTargetCmd(e)
	c.SetArgs([]string{"remove", "prod"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	cfg, _ = config.Load(e.Layout.Config)
	if len(cfg.Targets) != 0 {
		t.Fatal("not removed")
	}
	c = newTargetCmd(e)
	c.SetArgs([]string{"remove", "prod"})
	if err := c.Execute(); err == nil {
		t.Fatal("removing a missing target must fail")
	}
}
