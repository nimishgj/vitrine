package linux

import (
	"encoding/binary"

	"golang.org/x/net/bpf"
)

// Constants from <linux/seccomp.h>, <linux/audit.h>, <asm/ioctls.h>.
const (
	seccompRetAllow  = 0x7fff0000
	seccompRetErrno  = 0x00050000
	eperm            = 1
	auditArchX86_64  = 0xc000003e
	auditArchAARCH64 = 0xc00000b7
	tiocsti          = 0x5412
	sysIoctlX86_64   = 16
	sysIoctlAARCH64  = 29
)

// seccomp_data offsets: nr @0, arch @4, args[1] low word @ 16+8 = 24.
const (
	offNr   = 0
	offArch = 4
	offArg1 = 24
)

// SeccompProgram returns a raw BPF filter that returns EPERM for
// ioctl(fd, TIOCSTI, ...) on x86_64 and aarch64 and allows everything else.
func SeccompProgram() []byte {
	// Instruction indexes are noted so the skip counts can be checked:
	// a JumpIf at index i with Skip n lands at i+1+n.
	prog := []bpf.Instruction{
		/* 0 */ bpf.LoadAbsolute{Off: offArch, Size: 4},
		/* 1 */ bpf.JumpIf{Cond: bpf.JumpEqual, Val: auditArchX86_64, SkipTrue: 0, SkipFalse: 3}, // else -> 5
		/* 2 */ bpf.LoadAbsolute{Off: offNr, Size: 4},
		/* 3 */ bpf.JumpIf{Cond: bpf.JumpEqual, Val: sysIoctlX86_64, SkipTrue: 5, SkipFalse: 0}, // ioctl -> 9
		/* 4 */ bpf.RetConstant{Val: seccompRetAllow},
		/* 5 */ bpf.JumpIf{Cond: bpf.JumpEqual, Val: auditArchAARCH64, SkipTrue: 0, SkipFalse: 2}, // unknown arch -> 8
		/* 6 */ bpf.LoadAbsolute{Off: offNr, Size: 4},
		/* 7 */ bpf.JumpIf{Cond: bpf.JumpEqual, Val: sysIoctlAARCH64, SkipTrue: 1, SkipFalse: 0}, // ioctl -> 9
		/* 8 */ bpf.RetConstant{Val: seccompRetAllow},
		/* 9 */ bpf.LoadAbsolute{Off: offArg1, Size: 4},
		/* 10 */ bpf.JumpIf{Cond: bpf.JumpEqual, Val: tiocsti, SkipTrue: 0, SkipFalse: 1}, // not TIOCSTI -> 12
		/* 11 */ bpf.RetConstant{Val: seccompRetErrno | eperm},
		/* 12 */ bpf.RetConstant{Val: seccompRetAllow},
	}
	raw, err := bpf.Assemble(prog)
	if err != nil {
		panic(err) // static program; cannot fail
	}
	out := make([]byte, 0, len(raw)*8)
	for _, ins := range raw {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint16(b[0:], ins.Op)
		b[2] = ins.Jt
		b[3] = ins.Jf
		binary.LittleEndian.PutUint32(b[4:], ins.K)
		out = append(out, b...)
	}
	return out
}
