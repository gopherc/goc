// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

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

#include <stdbool.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>
#include <stdint.h>
#include <math.h>
#include <string.h>

#include <wasm-rt.h>

#define MAX_ARGC 256
#define MAX_FILES 256

#define LOAD(addr, ty) (*(ty*)(Z_mem->data + (addr)))
#define STORE(addr, ty, v) (*(ty*)(Z_mem->data + (addr)) = (v))

#define IMPL(name) \
    static void impl_ ## name (uint32_t sp); \
    void (*name)(uint32_t) = impl_ ## name; \
    static void impl_ ## name (uint32_t sp)

#define NOTIMPL(name) IMPL(name) { (void)sp; panic("not implemented: " #name); }

enum {
    O_RDONLY = 0x0,
    O_WRONLY = 0x1,
    O_RDWR   = 0x2,
    O_CREAT  = 0x40,
    O_TRUNC  = 0x200,
    O_APPEND = 0x400,
};

int descriptor_free_index = 3;
int descriptor_free_list[MAX_FILES] = {0};
FILE *descriptors[MAX_FILES] = {0};

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

static void panic(const char *s) {
    fprintf(stderr, "%s\n", s);
    exit(-1);
}

static int32_t write(int32_t sp) {
    int64_t fd = LOAD(sp+8, int64_t);
	int64_t p = LOAD(sp+16, int64_t);
	int32_t n = LOAD(sp+24, int32_t);

    if (fd < 0 || fd >= MAX_FILES)
        return -1;

    FILE *fp = descriptors[fd];
    if (!fp)
        return -1;
    
    return (int32_t)fwrite(&Z_mem->data[p], 1, (size_t)n, fp);
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
    int32_t n = (int32_t)write(sp);
    STORE(sp+32, int32_t, n);
}

/* import: 'go' 'syscall.readFile' */
IMPL(Z_goZ_syscallZ2EreadFileZ_vi) {
    int64_t fd = LOAD(sp+8, int64_t);
	int64_t p = LOAD(sp+16, int64_t);
	int32_t n = LOAD(sp+24, int32_t);

    if (fd < 0 || fd >= MAX_FILES)
        goto error;

    FILE *fp = descriptors[fd];
    if (!fp)
        goto error;

    if (feof(fp) != 0) {
        STORE(sp+32, int32_t, 0);
        STORE(sp+40, int32_t, 1);
        return;
    }

    int32_t r = (int32_t)fread(&Z_mem->data[p], 1, (size_t)n, fp);
    STORE(sp+32, int32_t, r);
    if (r == n || feof(fp) != 0) {
        STORE(sp+40, int32_t, 0);
        return;
    }
    STORE(sp+40, int32_t, -1);
    return;

error:
    STORE(sp+32, int32_t, 0);
    STORE(sp+40, int32_t, -1);
}

IMPL(Z_goZ_syscallZ2EcloseFileZ_vi) {
    int64_t fd = LOAD(sp+8, int64_t);
    if (fd < 3 || fd >= MAX_FILES || !descriptors[fd]) {
        STORE(sp+8, int32_t, -1);
        return;
    }

    fclose(descriptors[fd]);
    descriptors[fd] = NULL;
    descriptor_free_list[--descriptor_free_index] = fd;
    STORE(sp+8, int32_t, 0);
}

IMPL(Z_goZ_syscallZ2EopenFileZ_vi) {
    int64_t addr = LOAD(sp+8, int64_t);
    int64_t len = LOAD(sp+16, int64_t);
    uint8_t *ptr = &Z_mem->data[addr];

    int32_t mode = LOAD(sp+20, int32_t);
    int32_t perm = LOAD(sp+24, int32_t);

    char *tmp = malloc((size_t)len+1);
    memcpy(tmp, (void*)ptr, len);
    tmp[len] = 0;

    const char *access = NULL;
    switch (perm) {
    case O_RDONLY:
        access = "rb";
        break;
    case O_WRONLY | O_CREAT | O_TRUNC:
        access = "wb";
        break;
    case O_WRONLY | O_CREAT | O_APPEND:
        access = "ab";
        break;
    case O_RDWR:
        access = "rb+";
        break;
    case O_RDWR | O_CREAT | O_TRUNC:
        access = "wb+";
        break;
    case O_RDWR | O_CREAT | O_APPEND:
        access = "ab+";
        break;
    default:
        goto error;
    }

    FILE *fp = fopen(tmp, access);
    free(tmp);

    if (fp == 0 || descriptor_free_index == MAX_FILES - 1)
        goto error;

    int fd = descriptor_free_list[descriptor_free_index++];
    descriptors[fd] = fp;
    STORE(sp+32, int64_t, (int64_t)fd);
    return;

error:
    STORE(sp+32, int64_t, (int64_t)-1);
}

IMPL(Z_goZ_syscallZ2EflushFileZ_vi) {
    int64_t fd = LOAD(sp+8, int64_t);
    if (fd < 0 || fd >= MAX_FILES || !descriptors[fd]) {
        STORE(sp+16, int64_t, -1);
        return;
    }
    STORE(sp+16, int32_t, (int32_t)fflush(descriptors[fd]));
}

IMPL(Z_goZ_syscallZ2EtellFileZ_vi) {
    int64_t fd = LOAD(sp+8, int64_t);
    if (fd < 3 || fd >= MAX_FILES || !descriptors[fd]) {
        STORE(sp+16, int64_t, -1);
        return;
    }
    STORE(sp+16, int64_t, (int64_t)ftell(descriptors[fd]));
}

IMPL(Z_goZ_syscallZ2EseekFileZ_vi) {
    int64_t fd = LOAD(sp+8, int64_t);
    int64_t offset = LOAD(sp+16, int64_t);
    int32_t whence = LOAD(sp+24, int32_t);

    if (fd < 3 || fd >= MAX_FILES || !descriptors[fd])
        goto error;

    int origin;
    switch (whence) {
    case 0:
        origin = SEEK_SET;
        break;
    case 1:
        origin = SEEK_CUR;
        break;
    case 2:
        origin = SEEK_END;
        break;
    default:
        goto error;
    }

    STORE(sp+32, int32_t, (int32_t)fseek(descriptors[fd], (long)offset, origin));
    return;

error:
    STORE(sp+32, int32_t, -1);
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

int GOC_ENTRY(int argc, char *argv[]) {
    if (argc > MAX_ARGC)
        argc = MAX_ARGC;

    descriptors[0] = stdin;
    descriptors[1] = stdout;
    descriptors[2] = stderr;

    for (int i = 0; i < MAX_FILES; i++)
        descriptor_free_list[i] = i;

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