package sandbox

import "testing"

func TestBuildEnvAllowlist(t *testing.T) {
	host := []string{
		"PATH=/home/u/.vitrine/bin:/usr/bin:/bin",
		"TERM=xterm-256color", "LANG=en_US.UTF-8", "LC_ALL=C", "COLORTERM=truecolor",
		"AWS_SECRET_ACCESS_KEY=leak", "ANTHROPIC_API_KEY=leak", "HOME=/home/u", "KUBECONFIG=/home/u/.kube/config",
	}
	env := BuildEnv(EnvInput{
		HostEnv: host, ShimDir: "/home/u/.vitrine/bin",
		Home: "/agent/home", Scratch: "/scratch",
		Extra:   map[string]string{"KUBECONFIG": "/creds/kubeconfig"},
		GitName: "Nimisha", GitEmail: "n@example.com",
	})
	for _, k := range []string{"AWS_SECRET_ACCESS_KEY", "ANTHROPIC_API_KEY"} {
		if _, ok := env[k]; ok {
			t.Fatalf("%s leaked", k)
		}
	}
	if env["PATH"] != "/usr/bin:/bin" {
		t.Fatalf("PATH %q", env["PATH"])
	}
	if env["HOME"] != "/agent/home" || env["TMPDIR"] != "/scratch" || env["VITRINE"] != "1" {
		t.Fatalf("home/tmp: %v", env)
	}
	if env["TERM"] != "xterm-256color" || env["LC_ALL"] != "C" || env["LANG"] == "" || env["COLORTERM"] == "" {
		t.Fatal("terminal vars")
	}
	if env["KUBECONFIG"] != "/creds/kubeconfig" {
		t.Fatal("extra must override host")
	}
	if env["GIT_AUTHOR_NAME"] != "Nimisha" || env["GIT_COMMITTER_EMAIL"] != "n@example.com" {
		t.Fatal("git identity")
	}
}

func TestGitIdentity(t *testing.T) {
	fake := func(args ...string) (string, error) {
		switch args[len(args)-1] {
		case "user.name":
			return "N\n", nil
		default:
			return "", nil
		}
	}
	n, e := GitIdentity(fake)
	if n != "N" || e != "" {
		t.Fatal(n, e)
	}
}

func TestEnvSliceSorted(t *testing.T) {
	got := EnvSlice(map[string]string{"B": "2", "A": "1"})
	if len(got) != 2 || got[0] != "A=1" || got[1] != "B=2" {
		t.Fatal(got)
	}
}
