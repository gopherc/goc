// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

#ifdef __cplusplus
extern "C" {
#endif

#ifdef _WIN32
    #ifdef _CRT_SECURE_NO_WARNINGS
        #undef _CRT_SECURE_NO_WARNINGS
    #endif
    #define _CRT_SECURE_NO_WARNINGS 1

    #define WIN32_LEAN_AND_MEAN
    #include <windows.h>
#else
    extern char **environ;
#endif

#ifdef _MSC_VER
    #define EXPORT __declspec(dllexport)
#else
    #define EXPORT
#endif

#include <stdbool.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>
#include <stdint.h>
#include <math.h>
#include <string.h>

#include <wasm-rt.h>

#ifndef GOC_FWRITE
    #define GOC_FWRITE fwrite
#endif
extern size_t GOC_FWRITE(const void*, size_t, size_t, FILE*);

#ifndef GOC_ALLOC
    #define GOC_ALLOC realloc
#endif
extern void *GOC_ALLOC(void*, size_t);

#ifndef GOC_ENTRY
    #define GOC_ENTRY main
#endif

#define MAX_ARGC 256
#define PAGE_SIZE 65536

#define LOAD(addr, ty) (*(ty*)(Z_mem->data + (addr)))
#define STORE(addr, ty, v) (*(ty*)(Z_mem->data + (addr)) = (v))

#define IMPL(name) \
    static void impl_ ## name (uint32_t sp); \
    void (*name)(uint32_t) = impl_ ## name; \
    static void impl_ ## name (uint32_t sp)

#define NOTIMPL(name) IMPL(name) { (void)sp; panic("not implemented: " #name); }

int32_t exit_code = -1;
int32_t has_exit = 0;

/* export: 'run' */
extern void (*Z_runZ_vii)(uint32_t, uint32_t);
/* export: 'resume' */
extern void (*Z_resumeZ_vv)();
/* export: 'getsp' */
extern uint32_t (*Z_getspZ_iv)();
/* export: 'mem' */
extern wasm_rt_memory_t *Z_mem;

static void panic(const char *s) {
    fprintf(stderr, "%s\n", s);
    exit(-1);
}

static int32_t write(int32_t sp) {
    int64_t fd = LOAD(sp+8, int64_t);
	int64_t p = LOAD(sp+16, int64_t);
	int32_t n = LOAD(sp+24, int32_t);

    FILE *fp = NULL;
    if (fd == 1)
        fp = stdout;
    else if (fd == 2)
        fp = stderr;
    else
        return -1;
    
    return (int32_t)GOC_FWRITE(&Z_mem->data[p], 1, (size_t)n, fp);
}

uint32_t wasm_rt_call_stack_depth;

void wasm_rt_trap(wasm_rt_trap_t code) {
    panic("trap!");
}

uint32_t wasm_rt_register_func_type(uint32_t param_count, uint32_t result_count, ...) {
    return 0;
}

void wasm_rt_allocate_memory(wasm_rt_memory_t* memory, uint32_t initial_pages, uint32_t max_pages) {
    memory->pages = initial_pages;
    memory->max_pages = max_pages;
    memory->size = initial_pages * PAGE_SIZE;
    memory->data = GOC_ALLOC(NULL, memory->size);
    memset(memory->data, 0, memory->size);
}

uint32_t wasm_rt_grow_memory(wasm_rt_memory_t* memory, uint32_t delta) {
    uint32_t old_pages = memory->pages;
    uint32_t new_pages = memory->pages + delta;
    if (new_pages < old_pages || new_pages > memory->max_pages)
        return (uint32_t)-1;

    memory->pages = new_pages;
    memory->size = new_pages * PAGE_SIZE;
    memory->data = GOC_ALLOC(memory->data, memory->size);
    memset(memory->data + old_pages * PAGE_SIZE, 0, delta * PAGE_SIZE);
    return old_pages;
}

void wasm_rt_allocate_table(wasm_rt_table_t* table, uint32_t elements, uint32_t max_elements) {
    table->size = elements;
    table->max_size = max_elements;
    table->data = GOC_ALLOC(NULL, table->size * sizeof(wasm_rt_elem_t));
    memset(table->data, 0, table->size * sizeof(wasm_rt_elem_t));
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
    int64_t nanotime = (int64_t)(((double)clock() / CLOCKS_PER_SEC) * 1000000000);
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

    static int32_t time_id_counter = 0;
    STORE(sp+16, int32_t, time_id_counter++);
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
    int32_t n = write(sp);
    STORE(sp+32, int32_t, n);
}

/* import: 'go' 'syscall.readFile' */
IMPL(Z_goZ_syscallZ2EreadFileZ_vi) {
    int64_t fd = LOAD(sp+8, int64_t);
	int64_t p = LOAD(sp+16, int64_t);
	int32_t n = LOAD(sp+24, int32_t);

    if (fd != 0)
        goto error;

    if (feof(stdin) != 0) {
        STORE(sp+32, int32_t, 0);
        STORE(sp+40, int32_t, 1);
        return;
    }

    int32_t r = (int32_t)fread(&Z_mem->data[p], 1, (size_t)n, stdin);
    STORE(sp+32, int32_t, r);
    if (r == n || feof(stdin) != 0) {
        STORE(sp+40, int32_t, 0);
        return;
    }
    STORE(sp+40, int32_t, -1);
    return;

error:
    STORE(sp+32, int32_t, 0);
    STORE(sp+40, int32_t, -1);
}

static int write_string(int *offset, const char *str) {
    int p = *offset;
    int ln = strlen(str);
    memcpy(&Z_mem->data[*offset], str, ln+1);
    *offset += ln + (8 - (ln % 8));
    return p;
}

extern void init();

int pointerOffsets[MAX_ARGC] = {0};

EXPORT int GOC_ENTRY(int argc, char *argv[]) {
    if (argc > MAX_ARGC)
        argc = MAX_ARGC;
    
    srand((unsigned)time(NULL));
    init();

    /* Pass command line arguments and environment variables by writing them to the linear memory. */
	int offset = 4096;
    int nPtr = 0;
    for (int i = 0; i < argc; i++)
        pointerOffsets[nPtr++] = write_string(&offset, argv[i]);

    /* Num env */
    int *count = &pointerOffsets[nPtr++];
    *count = 0;

    #ifdef _WIN32
        for (const char *envs = GetEnvironmentStringsA(); *envs; envs += strlen(envs) + 1) {
            pointerOffsets[nPtr++] = write_string(&offset, envs);
            (*count)++;
        }
    #else
        if (environ) {
            for (const char **envs = environ; *envs; envs++) {
                pointerOffsets[nPtr++] = write_string(&offset, *envs);
                (*count)++;
            }
        }
    #endif
    
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

#ifdef __cplusplus
}
#endif