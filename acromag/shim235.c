// shim contains a helper function that does type conversion
// within C to avoid making the Go type system angry
#include "apcommon.h"
#include "AP235.h"
#include <sys/mman.h>

APSTATUS GetAPAddress235(int nHandle, struct mapap235** pAddress)
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
unsigned long *Setup_board_corrected_buffer(struct cblk235 *cfg)
{
	unsigned long *scatter_info = malloc(4*sizeof(unsigned long));
	// unsigned long *scatter_info; /* scatter-gather input parameters, space for 4 parameters */
	// scatter_info = ptr;
	cfg->pcor_buf = aligned_malloc(sizeof(short[16][MAXSAMPLES]), ALIGNMENT); /* allocate DMA buffer */
	mlock(cfg->pcor_buf, sizeof(short[16][MAXSAMPLES]));						/* lock pages in memory */

	/* Map user pages and create scatter-gather list for DMA xfers */
	scatter_info[0] = (unsigned long)&cfg->pcor_buf[0][0]; /* users data buffer virtual address */
	/* users data buffer size (buffer space x 16 channels) */
	scatter_info[1] = (unsigned long)sizeof(cfg->pcor_buf[0][MAXSAMPLES]) * 16;
	/* external (PCI) address of the on board scatterlist RAM */
	scatter_info[2] = (unsigned long)&cfg->brd_ptr->CHAN[0].fptrlo.NxtDescPtrLo;
	scatter_info[3] = (unsigned long)cfg->pAP->nDevInstance; /* get board instance index */
	/* Map user pages and build scatter/gather list for DMA xfers */
	ioctl(cfg->pAP->nAPDeviceHandle, 8, &scatter_info[0]); /* function 8 builds scatter/gather list */
	cfg->bInitialized = TRUE;
	cfg->bAP = TRUE;

	memset(&cfg->IDbuf[0],0,sizeof(cfg->IDbuf));	/* empty the buffer */
    ReadFlashID235(cfg, &cfg->IDbuf[0]);

    if( (strstr( (const char *)&cfg->IDbuf[0], (const char *)"AP235" ) == NULL) )	{/* AP2X5 ID */
		  return NULL;
	}
    else
    {
		rcc235(cfg); /* read the calibration coef. into an array */
	}
	return scatter_info;
}

void Teardown_board_corrected_buffer(struct cblk235 *cfg)
{
	unsigned long scatter_info[4];
	scatter_info[0] = (unsigned long)cfg->pAP->nDevInstance; /* get board instance */
	ioctl(cfg->pAP->nAPDeviceHandle, 9, &scatter_info[0]);   /* unmap user pages and scatter-gather list */

	munlock(cfg->pcor_buf, sizeof(short[16][MAXSAMPLES])); /* unlock pages in memory */
	aligned_free((void *)cfg->pcor_buf);					 /* free allocated DMA buffer on exit */
}

short* MkDataArray(int size)
{
	short *array = calloc(size, sizeof(short));
	if (array == NULL) {
		return NULL;
	}
	return array;
}

void enable_interrupts(struct cblk235 *cfg)
{
	output_long(cfg->nHandle, (long*)&cfg->brd_ptr->AXI_MasterEnableRegister, MasterInterruptEnable);
}

void start_waveform(struct cblk235 *cfg)
{
	long temp = input_long(cfg->nHandle, (long *)&cfg->brd_ptr->CommonControl);
	temp |= 1;	/* Start All Waveforms */
	output_long(cfg->nHandle, (long *)&cfg->brd_ptr->CommonControl, (long)temp);
}

void stop_waveform(struct cblk235 *cfg)
{
	// long temp = input_long(cfg->nHandle, (long *)&cfg->brd_ptr->CommonControl);
	// temp &= 0xFFFFFFFE;	/* Stop All Waveforms */
	// output_long(cfg->nHandle, (long *)&cfg->brd_ptr->CommonControl, (long)temp);
	output_long(cfg->nHandle, (long *)(&cfg->brd_ptr->CommonControl), (long)(0x10));

	// disable interrupts
	output_long(cfg->nHandle, (long *)(&cfg->brd_ptr->AXI_ClearInterruptEnableRegister), (long)(0x1FFFF));
	output_long(cfg->nHandle, (long *)(&cfg->brd_ptr->AXI_MasterEnableRegister), (long)(MasterInterruptDisable));
	APTerminateBlockedStart(cfg->nHandle);
}

unsigned long fetch_status(struct cblk235 *cfg)
{

	return APBlockingStartConvert(cfg->nHandle, (long *)(&cfg->brd_ptr->AXI_MasterEnableRegister), (long)(MasterInterruptEnable), (long)(2));
}

void refresh_interrupt(struct cblk235 *cfg, unsigned long status)
{
	// ACK the interrupt
	output_long(cfg->nHandle, (long *)(&cfg->brd_ptr->AXI_InterruptAcknowledgeRegister), (long)(status&0xFFFF));

	// re-enable interrupt
	output_long(cfg->nHandle, (long *)(&cfg->brd_ptr->AXI_SetInterruptEnableRegister), (long)(status&0xFFFF));
}

void set_DAC_sample_addresses(struct cblk235 *cfg, int channel)
{
	output_long(cfg->nHandle, (long *)&cfg->brd_ptr->DAC[channel].StartAddr, (long)channel * MAXSAMPLES);
	output_long(cfg->nHandle, (long *)&cfg->brd_ptr->DAC[channel].EndAddr, (long)channel * MAXSAMPLES + 4095);
}
