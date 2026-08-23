#ifndef GO_TEST_MAIN_PACKAGE
#error GO_TEST_MAIN_PACKAGE must name the generated Go test main package
#endif

#include <string.h>

#ifdef __APPLE__
#define GO_SYMBOL(name) __asm__("_" name)
#else
#define GO_SYMBOL(name) __asm__(name)
#endif

extern void llgo_test_init(void) GO_SYMBOL(GO_TEST_MAIN_PACKAGE ".init");
extern void llgo_test_run(void) GO_SYMBOL(GO_TEST_MAIN_PACKAGE ".main");
extern int __llgo_argc;
extern char **__llgo_argv;

int main(int argc, char **argv) {
#ifdef GO_C_SHARED
    if (__llgo_argc != argc || __llgo_argv == NULL) {
        return 101;
    }
    for (int i = 0; i < argc; i++) {
        if (__llgo_argv[i] == NULL || strcmp(__llgo_argv[i], argv[i]) != 0) {
            return 101;
        }
    }
#else
    __llgo_argc = argc;
    __llgo_argv = argv;
#endif
    llgo_test_init();
    llgo_test_run();
    return 0;
}
