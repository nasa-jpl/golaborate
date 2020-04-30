
/*
{+D}
    SYSTEM:	    Pentium

    FILENAME:	    apcommon.h

    MODULE NAME:    Functions common to the Pentium example software.

    VERSION:	    B

    CREATION DATE:  12/01/15

    DESIGNED BY:    FJM

    CODED BY:	    FJM

    ABSTRACT:       This file contains the definitions, structures and prototypes.

    CALLING
	SEQUENCE:

    MODULE TYPE:

    I/O RESOURCES:

    SYSTEM
	RESOURCES:

    MODULES
	CALLED:

    REVISIONS:

  DATE	   BY	    PURPOSE
--------  ----	------------------------------------------------
02/01/17  FJM   Added APTerminateBlockedStart() and modified APblocking_start_convert()

{-D}
*/


/*
    MODULES FUNCTIONAL DETAILS:

	This file contains the definitions, structures and prototypes.
*/

#define VENDOR_ID (word)0x16D5		/* Acromag's vendor ID for all PCI bus products */
#define MAX_APS 4			/* maximum number of boards */


#ifndef BUILDING_FOR_KERNEL
#include <stdio.h>
#include <sys/types.h>
#include <fcntl.h>
#include <inttypes.h>
#include <unistd.h>
#include <sys/ioctl.h>

/* Required for FC4 */
#include <stdlib.h>     /* malloc */
#include <string.h>     /* memset */
#endif /* BUILDING_FOR_KERNEL */



typedef unsigned char BYTE;
typedef int BOOL;
typedef unsigned char byte;		/* custom data type */
typedef unsigned short word;		/* custom data type */
typedef short WORD;



typedef int APSTATUS;			/* custom made APSTATUS data type, used as a
					   return value from the common functions. */
#define TRUE	1			/* Boolean value for true */
#define FALSE	0			/* Boolean value for false */



#define AP_RESET	0x8000	/* ORed with interrupt register to reset board */
#define AP_INT_ENABLE	0x0001	/* interrupt enable */
#define AP_INT_PENDING	0x0002	/* interrupt pending bit */



/*
	APSTATUS return values
	Errors will have most significant bit set and are preceded with an E_.
	Success values will be succeeded with an S_.
*/

#define ERROR			0x8000	/* general */
#define E_OUT_OF_MEMORY 	0x8001	/* Out of memory status value */
#define E_OUT_OF_APS	   	0x8002	/* All AP spots have been taken */
#define E_INVALID_HANDLE	0x8003	/* no AP exists for this handle */
#define E_NOT_INITIALIZED	0x8006	/* Pmc not initialized */
#define E_NOT_IMPLEMENTED	0x8007	/* Function is not implemented */
#define E_NO_INTERRUPTS 	0x8008	/* unable to handle interrupts */
#define S_OK			0x0000	/* Everything worked successfully */


/*
	AP data structure
*/

typedef struct
{
	int nHandle;			/* handle from addpmc() */
	int nAPDeviceHandle;		/* handle from kernel open() */
	long lBaseAddress;		/* pointer to base address of board */
	int nDevInstance;		/* Device Instance */
	int nInteruptID;		/* ID of interrupt handler */
	int nIntLevel;			/* Interrupt level */
	char devname[64];		/* device name */
	BOOL bInitialized;		/* intialized flag */
	BOOL bIntEnabled;		/* interrupts enabled flag */
}APDATA_STRUCT;

typedef struct
{
	uint32_t InterruptRegister;	/* Interrupt Pending/control Register */
}AP_BOARD_MEMORY_MAP;

/*
	Function Prototypes
*/

APSTATUS GetAPAddress(int nHandle, long* pAddress);
APSTATUS SetAPAddress(int nHandle, long lAddress);
APSTATUS EnableAPInterrupts(int nHandle);
APSTATUS DisableAPInterrupts(int nHandle);
APSTATUS InitAPLib(void);
APSTATUS APOpen(int nDevInstance, int* pHandle, char* devname);
APSTATUS APClose(int nHandle);
APSTATUS APInitialize(int nHandle);


word SwapBytes( word v );
long SwapLong( long v );


/*  Functions used by above functions */
void AddAP(APDATA_STRUCT* pAP);
void DeleteAP(int nHandle);
APDATA_STRUCT* GetAP(int nHandle);
byte input_byte(int nHandle, byte*);		/* function to read an input byte */
word input_word(int nHandle, word*);		/* function to read an input word */
void output_byte(int nHandle, byte*, byte);	/* function to output a byte */
void output_word(int nHandle, word*, word);	/* function to output a word */
long input_long(int nHandle, long*);		/* function to read an input long */
void output_long(int nHandle, long*, long);	/* function to output a long */
uint32_t APBlockingStartConvert(int nHandle, long *p, long v, long parameter);
void APTerminateBlockedStart(int nHandle);
long get_param(void);		/* input a parameter */

