// Generated by the GopherC bind tool.
// 2019-04-04 21:04:36.8755411 +0200 CEST m=+0.003001801

#include <string.h>
#include <wasm-rt.h>

extern uint32_t (*Z_getspZ_iv)();
extern wasm_rt_memory_t *Z_mem;


extern int putchar(int);
static void _Z_goZ_githubZ2EcomZ2FgophercZ2FgocZ2FtestsZ2FbindZ2FbindZ2EgocPutcZ_vi(uint32_t sp) {
	sp += 8;
	int _ch = *(int*)&Z_mem->data[sp];
	sp += sizeof(int) + ((8 - (sizeof(int) % 8)) % 8);
	int _r = putchar(_ch);
	memcpy(&Z_mem->data[sp], &_r, sizeof(int));
}
void (*Z_goZ_githubZ2EcomZ2FgophercZ2FgocZ2FtestsZ2FbindZ2FbindZ2EgocPutcZ_vi)(uint32_t) = _Z_goZ_githubZ2EcomZ2FgophercZ2FgocZ2FtestsZ2FbindZ2FbindZ2EgocPutcZ_vi;

