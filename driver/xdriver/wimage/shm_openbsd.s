//go:build openbsd && (amd64 || arm64)

#include "textflag.h"

TEXT shmget_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_wimage_shmget(SB)
GLOBL ·shmGetAddr(SB), RODATA, $8
DATA ·shmGetAddr(SB)/8, $shmget_trampoline<>(SB)

TEXT shmat_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_wimage_shmat(SB)
GLOBL ·shmAtAddr(SB), RODATA, $8
DATA ·shmAtAddr(SB)/8, $shmat_trampoline<>(SB)

TEXT shmdt_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_wimage_shmdt(SB)
GLOBL ·shmDtAddr(SB), RODATA, $8
DATA ·shmDtAddr(SB)/8, $shmdt_trampoline<>(SB)

TEXT shmctl_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_wimage_shmctl(SB)
GLOBL ·shmCtlAddr(SB), RODATA, $8
DATA ·shmCtlAddr(SB)/8, $shmctl_trampoline<>(SB)
