/*
Copyright (C) 2016-2019 Andreas T Jonsson

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

#include <stdlib.h>
#include <stdio.h>
#include <time.h>

#include <out.h>

#define LOAD(addr, ty) (*(ty*)(Z_mem->data + (addr)))
#define STORE(addr, ty, v) (*(ty*)(Z_mem->data + (addr)) = (v))

#define IMPL(name) \
    static void impl_ ## name (u32 sp); \
    void (*name)(u32) = impl_ ## name; \
    static void impl_ ## name (u32 sp)

#define NOTIMPL(name) IMPL(name) { (void)sp; panic("not implemented: " #name); }

FILE *iob[3] = {0};
s32 exit_code = 0;
s32 has_exit = 0;

void panic(const char *s) {
    fprintf(stderr, "%s\n", s);
    exit(-1);
}

/* import: 'go' 'debug' */
IMPL(Z_goZ_debugZ_vi) {
    printf("%d\n", sp);
}

/* import: 'go' 'runtime.wasmExit' */
IMPL(Z_goZ_runtimeZ2EwasmExitZ_vi) {
    exit_code = LOAD(sp+8, s32);
    has_exit = 1;
}

/* import: 'go' 'runtime.wasmWrite' */
IMPL(Z_goZ_runtimeZ2EwasmWriteZ_vi) {
    s64 fd = LOAD(sp+8, s64);
    if (fd != 1 && fd != 2)
        panic("invalid file descriptor");

	s64 p = LOAD(sp+16, s64);
	s32 n = LOAD(sp+24, s32);
    fwrite((void*)p, 1, (size_t)n, iob[fd]);
}

/* import: 'go' 'runtime.nanotime' */
IMPL(Z_goZ_runtimeZ2EnanotimeZ_vi) {
    s64 nanotime = (s64)(clock() / CLOCKS_PER_SEC) * 1000000000;
    STORE(sp+8, s64, nanotime);
}

/* import: 'go' 'runtime.walltime' */
NOTIMPL(Z_goZ_runtimeZ2EwalltimeZ_vi)

/* import: 'go' 'runtime.scheduleTimeoutEvent' */
NOTIMPL(Z_goZ_runtimeZ2EscheduleTimeoutEventZ_vi)

/* import: 'go' 'runtime.clearTimeoutEvent' */
NOTIMPL(Z_goZ_runtimeZ2EclearTimeoutEventZ_vi)

/* import: 'go' 'runtime.getRandomData' */
IMPL(Z_goZ_runtimeZ2EgetRandomDataZ_vi) {
    s64 start = LOAD(sp+8, s64);
    s64 len = LOAD(sp+16, s64);

    for (s64 i = 0; i < len; i++)
        Z_mem->data[start+i] = (uint8_t)(rand() % 256);
}

/* import: 'go' 'syscall/js.stringVal' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EstringValZ_vi)

/* import: 'go' 'syscall/js.valueGet' */
IMPL(Z_goZ_syscallZ2FjsZ2EvalueGetZ_vi) {
    /* Return nil */
    const u32 nanHead = 0x7FF80000;
    STORE(sp+32+4, u32, nanHead);
    STORE(sp+32, u32, 2);
}

/* import: 'go' 'syscall/js.valueSet' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvalueSetZ_vi)

/* import: 'go' 'syscall/js.valueIndex' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvalueIndexZ_vi)

/* import: 'go' 'syscall/js.valueSetIndex' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvalueSetIndexZ_vi)

/* import: 'go' 'syscall/js.valueCall' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvalueCallZ_vi)

/* import: 'go' 'syscall/js.valueNew' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvalueNewZ_vi)

/* import: 'go' 'syscall/js.valueLength' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvalueLengthZ_vi)

/* import: 'go' 'syscall/js.valuePrepareString' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvaluePrepareStringZ_vi)

/* import: 'go' 'syscall/js.valueLoadString' */
NOTIMPL(Z_goZ_syscallZ2FjsZ2EvalueLoadStringZ_vi)

int main(int argc, char *argv[]) {
    iob[0] = stdin; iob[1] = stdout; iob[2] = stderr;
    srand(time(NULL));
    init();
    Z_runZ_vii(0, 0);
    while (!has_exit)
        Z_resumeZ_vv();
    return exit_code;
}