// shim contains a helper function that does type conversion
// within C to avoid making the Go type system angry
#include "../apcommon/apcommon.h"
#include "AP235.h"
#include <sys/mman.h>

APSTATUS GetAPAddress2(int nHandle, struct mapap235** pAddress)
{
	return (APSTATUS)GetAPAddress(nHandle, (long*)pAddress);
}

// taken from acromag drvr235.c, L150-162
/* Memory allocation helper routine */
#define ALIGNMENT 1048576 /* selected to minimize translation register updates */

void *aligned_malloc(int size, int align)
{
	void *mem = malloc(size + align + sizeof(void *));
	void **ptr = (void **)((long)(mem + align + sizeof(void *)) & ~(align - 1));
	ptr[-1] = mem;
	return ptr;
}
/* Memory deallocation helper routine */
void aligned_free(void *ptr)
{
	free(((void **)ptr)[-1]);
}

// refactored/taken from acromag drvr235.c, L251-262
void Setup_board_corrected_buffer(struct cblk235* cfg)
{
	unsigned long scatter_info[4]; /* scatter-gather input parameters, space for 4 parameters */
	struct cblk235 cfg2;
	cfg2 = *cfg;
	cfg2.pcor_buf = aligned_malloc(sizeof(short[16][MAXSAMPLES]), ALIGNMENT); /* allocate DMA buffer */
	mlock(cfg2.pcor_buf, sizeof(short[16][MAXSAMPLES]));						/* lock pages in memory */

	/* Map user pages and create scatter-gather list for DMA xfers */
	scatter_info[0] = (unsigned long)&cfg2.pcor_buf[0][0]; /* users data buffer virtual address */
	/* users data buffer size (buffer space x 16 channels) */
	scatter_info[1] = (unsigned long)sizeof(cfg2.pcor_buf[0][MAXSAMPLES]) * 16;
	/* external (PCI) address of the on board scatterlist RAM */
	scatter_info[2] = (unsigned long)&cfg2.brd_ptr->CHAN[0].fptrlo.NxtDescPtrLo;
	scatter_info[3] = (unsigned long)cfg2.pAP->nDevInstance; /* get board instance index */
	/* Map user pages and build scatter/gather list for DMA xfers */
	ioctl(cfg2.pAP->nAPDeviceHandle, 8, &scatter_info[0]); /* function 8 builds scatter/gather list */
	cfg2.bInitialized = TRUE;
	cfg2.bAP = TRUE;
}
