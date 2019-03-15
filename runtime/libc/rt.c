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
#include <stdint.h>
#include <math.h>
#include <string.h>

#include <wasm-rt.h>

#define MAX_ARGC 256

#define LOAD(addr, ty) (*(ty*)(Z_mem->data + (addr)))
#define STORE(addr, ty, v) (*(ty*)(Z_mem->data + (addr)) = (v))

#define IMPL(name) \
    static void impl_ ## name (uint32_t sp); \
    void (*name)(uint32_t) = impl_ ## name; \
    static void impl_ ## name (uint32_t sp)

#define NOTIMPL(name) IMPL(name) { (void)sp; panic("not implemented: " #name); }

FILE *iob[3] = {0};
int32_t exit_code = 0;
int32_t has_exit = 0;

/* export: 'run' */
extern void (*Z_runZ_vii)(uint32_t, uint32_t);
/* export: 'resume' */
extern void (*Z_resumeZ_vv)();
/* export: 'getsp' */
extern uint32_t (*Z_getspZ_iv)();
/* export: 'mem' */
extern wasm_rt_memory_t *Z_mem;

void panic(const char *s) {
    fprintf(stderr, "%s\n", s);
    exit(-1);
}

size_t write(int32_t sp) {
    int64_t fd = LOAD(sp+8, int64_t);
    if (fd != 1 && fd != 2)
        panic("invalid file descriptor");

	int64_t p = LOAD(sp+16, int64_t);
	int32_t n = LOAD(sp+24, int32_t);
    return fwrite(&Z_mem->data[p], 1, (size_t)n, iob[fd]);
}

/* import: 'go' 'debug' */
IMPL(Z_goZ_debugZ_vi) {
    printf("%d\n", sp);
}

/* import: 'go' 'runtime.wasmExit' */
IMPL(Z_goZ_runtimeZ2EwasmExitZ_vi) {
    exit_code = LOAD(sp+8, int32_t);
    has_exit = 1;
}

/* import: 'go' 'runtime.wasmWrite' */
IMPL(Z_goZ_runtimeZ2EwasmWriteZ_vi) {
    write(sp);
}

/* import: 'go' 'runtime.nanotime' */
IMPL(Z_goZ_runtimeZ2EnanotimeZ_vi) {
    int64_t nanotime = (int64_t)(clock() / CLOCKS_PER_SEC) * 1000000000;
    /* Avoid returning 0 so we add 1. */
    STORE(sp+8, int64_t, nanotime+1);
}

/* import: 'go' 'runtime.walltime' */
IMPL(Z_goZ_runtimeZ2EwalltimeZ_vi) {
    struct tm y2k = {0};
    y2k.tm_year = 70; y2k.tm_mday = 1;

    time_t timer;
    time(&timer);

    double seconds = difftime(timer, mktime(&y2k));
    STORE(sp+8, int64_t, (int64_t)seconds);

    double ipart;
    STORE(sp+16, int32_t, (int32_t)(modf(seconds, &ipart) * 1000000000));
}

/* import: 'go' 'runtime.scheduleTimeoutEvent' */
IMPL(Z_goZ_runtimeZ2EscheduleTimeoutEventZ_vi) {
    int64_t t = LOAD(sp+8, int64_t);
    (void)t;
    STORE(sp+16, int32_t, 0 /* id */);
}

/* import: 'go' 'runtime.clearTimeoutEvent' */
IMPL(Z_goZ_runtimeZ2EclearTimeoutEventZ_vi) {
    int32_t id = LOAD(sp+8, int32_t);
    (void)id;
}

/* import: 'go' 'runtime.getRandomData' */
IMPL(Z_goZ_runtimeZ2EgetRandomDataZ_vi) {
    int64_t start = LOAD(sp+8, int64_t);
    int64_t len = LOAD(sp+16, int64_t);

    for (int64_t i = 0; i < len; i++)
        Z_mem->data[start+i] = (uint8_t)(rand() % 256);
}

/* import: 'go' 'crypto/rand.getRandomValues' */
IMPL(Z_goZ_cryptoZ2FrandZ2EgetRandomValuesZ_vi) {
    Z_goZ_runtimeZ2EgetRandomDataZ_vi(sp);
}

/* import: 'go' 'syscall.writeFile' */
IMPL(Z_goZ_syscallZ2EwriteFileZ_vi) {
    int32_t n = (int32_t)write(sp);
    STORE(sp+32, int32_t, n);
}

/* import: 'go' 'syscall.readFile' */
IMPL(Z_goZ_syscallZ2EreadFileZ_vi) {
    int64_t fd = LOAD(sp+8, int64_t);
    if (fd != 0)
        panic("invalid file descriptor");

	int64_t p = LOAD(sp+16, int64_t);
	int32_t n = LOAD(sp+24, int32_t);
    int32_t r = (int32_t)fread(&Z_mem->data[p], 1, (size_t)n, iob[fd]);
    STORE(sp+32, int32_t, r);
}

int write_string(int *offset, const char *str) {
    int p = *offset;
    int ln = strlen(str);
    memcpy(&Z_mem->data[*offset], str, ln+1);
    *offset += ln + (8 - (ln % 8));
    return p;
}

extern void init();

int pointerOffsets[MAX_ARGC] = {0};

int GOC_ENTRY(int argc, char *argv[]) {
    if (argc > MAX_ARGC) {
        argc = MAX_ARGC;
    }

    iob[0] = stdin; iob[1] = stdout; iob[2] = stderr;
    srand((unsigned)time(NULL));
    init();

    /* Pass command line arguments and environment variables by writing them to the linear memory. */
	int offset = 4096;
    int nPtr = 0;
    for (int i = 0; i < argc; i++)
        pointerOffsets[nPtr++] = write_string(&offset, argv[i]);

    /* Num env */
    pointerOffsets[nPtr++] = 0;

    /* Should push environment variables here. */
    
    uint32_t argvOffset = (uint32_t)offset;
    for (int i = 0; i < nPtr; i++) {
        STORE(offset, uint32_t, pointerOffsets[i]);
        STORE(offset+4, uint32_t, 0);
        offset += 8;
    }

    Z_runZ_vii((uint32_t)argc, argvOffset);
    while (!has_exit)
        Z_resumeZ_vv();
    return exit_code;
}