#ifndef DEVICE_NAME
#include "../apcommon/apcommon.h"
#include "AP235.h"
#endif
APSTATUS GetAPAddress2(int nhandle, struct mapap235** addr);

int Setup_board_corrected_buffer(struct cblk235 *cfg);

void Teardown_board_corrected_buffer(struct cblk235 *cfg);

short* MkDataArray(int size);

void* aligned_malloc(int size, int align);

void aligned_free(void *ptr);
