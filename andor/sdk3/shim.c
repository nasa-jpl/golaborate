#include <stdlib.h>

// suffix 2 is to avoid clash with aligned_malloc from ap235.go, which likely
// will be built into the same binary.  This is copy pasted from there.
void *aligned_malloc2(int size, int align)
{
    void *mem = malloc(size + align + sizeof(void *));
    void **ptr = (void **)((long)(mem + align + sizeof(void *)) & ~(align - 1));
    ptr[-1] = mem;
    return ptr;
}
/* Memory deallocation helper routine */
void aligned_free2(void *ptr)
{
    free(((void **)ptr)[-1]);
}
