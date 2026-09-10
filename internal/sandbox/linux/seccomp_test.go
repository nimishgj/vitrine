package linux

import "testing"

func TestSeccompProgramShape(t *testing.T) {
	prog := SeccompProgram()
	if len(prog)%8 != 0 || len(prog) < 8*6 {
		t.Fatalf("program length %d", len(prog))
	}
}

func TestSeccompJumps(t *testing.T) {
	raw := SeccompProgram()
	// instruction 3: jt must land on the arg check (index 9)
	if jt := raw[3*8+2]; int(jt) != 5 {
		t.Fatalf("ins3 jt=%d want 5", jt)
	}
	// instruction 5: jf must land on allow (index 8)
	if jf := raw[5*8+3]; int(jf) != 2 {
		t.Fatalf("ins5 jf=%d want 2", jf)
	}
}
