
/*
{+D}
    SYSTEM:		Linux

    FILENAME:		apcommon.c

    MODULE NAME:	Functions common to the example software.

    VERSION:		B

    CREATION DATE:	12/01/15

    DESIGNED BY:	FJM

    CODED BY:		FJM

    ABSTRACT:		This file contains the implementation of the functions for modules.

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
02/01/17   FJM  Added function APTerminateBlockedStart() and modified APblocking_start_convert

{-D}
*/

/*
	This file contains the implementation of the functions for Acromag modules.
*/

#include "apcommon.h"

/*	Global variables */
int	gNumberAPs = -1;		/* Number of boards that have been opened and/or flag = -1...
	                               if library is uninitialized see function InitAPLib() */


APDATA_STRUCT *gpAP[MAX_APS];	/* pointer to the boards */



/*
        Some systems can resolve BIG_ENDIAN/LITTLE_ENDIAN data transfers in hardware.
        If the system is resolving BIG_ENDIAN/LITTLE_ENDIAN data transfers in hardware
        the SWAP_ENDIAN define should be commented out.

        When resolving the BIG_ENDIAN/LITTLE_ENDIAN data transfers in hardware is not
        possible or desired the SWAP_ENDIAN define is provided.

        Define SWAP_ENDIAN to enable software byte swapping for word and long transfers
*/

/* #define SWAP_ENDIAN		/ * SWAP_ENDIAN enables software byte swapping for word and long transfers */



/*	Use this define to exclude I/O functions when two or more common files are combined in the same project */

/*#define NO_AP_IOFUNCTIONS */

#ifndef NO_AP_IOFUNCTIONS
word SwapBytes( word v )
{
#ifdef SWAP_ENDIAN		/* endian correction if needed */
  word  Swapped;

  Swapped = v << 8;
  Swapped |= ( v >> 8 );
  return( Swapped );
#else				/* no endian correction needed */
  return( v );
#endif /* SWAP_ENDIAN */

}


long SwapLong( long v )
{
#ifdef SWAP_ENDIAN		/* endian correction if needed */
 word Swap1, Swap2;
 long Swapped;

  Swap1 = (word)(v >> 16);
  Swap1 = SwapBytes( Swap1 );

  Swap2 = (word)v & 0xffff;
  Swap2 = SwapBytes( Swap2 );

  Swapped = (long)(Swap2 << 16);
  Swapped |= (long)(Swap1 & 0xffff);
  return( Swapped );
#else				/* no endian correction needed */
  return( v );
#endif /* SWAP_ENDIAN */

}


byte input_byte(int nHandle, byte *p)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data[2];

	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return((byte)0);

	if( p )
	{
           /* place address to read byte from in data [0]; */
           data[0] = (unsigned long) p;
           data[1] = (unsigned long) 0;
           /* pram3 = function: 1=read8bits,2=read16bits,4=read32bits */
           read( pAP->nAPDeviceHandle, &data[0], 1 );
           return( (byte)data[1] );
	}
	return((byte)0);
}

word input_word(int nHandle, word *p)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data[2];

	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return((word)0);

	if( p )
	{
           /* place address to read word from in data [0]; */
           data[0] = (unsigned long) p;
           /* pram3 = function: 1=read8bits,2=read16bits,4=read32bits */
           read( pAP->nAPDeviceHandle, &data[0], 2 );
           return(  SwapBytes( (word)data[1] ) );
	}
	return((word)0);
}


long input_long(int nHandle, long *p)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data[2];

	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return((long)0);

	if( p )
	{
           /* place address to read word from in data [0]; */
           data[0] = (unsigned long) p;
           /* pram3 = function: 1=read8bits,2=read16bits,4=read32bits */
           read( pAP->nAPDeviceHandle, &data[0], 4 );
           return(  SwapLong( (long)data[1] ) );
	}
	return((long)0);
}


void output_byte(int nHandle, byte *p, byte v)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data[2];

	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return;

	if( p )
	{
		/* place address to write byte in data [0]; */
		data[0] = (unsigned long) p;
		/* place value to write @ address data [1]; */
		data[1] = (unsigned long) v;
	        /* pram3 = function: 1=write8bits,2=write16bits,4=write32bits */
		write( pAP->nAPDeviceHandle, &data[0], 1 );
	}
}

void output_word(int nHandle, word *p, word v)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data[2];

	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return;

	if( p )
	{
           /* place address to write word in data [0]; */
           data[0] = (unsigned long) p;
           /* place value to write @ address data [1]; */
           data[1] = (unsigned long) SwapBytes( v );
           /* pram3 = function: 1=write8bits,2=write16bits,4=write32bits */
           write( pAP->nAPDeviceHandle, &data[0], 2 );
	}
}


void output_long(int nHandle, long *p, long v)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data[2];

	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return;

	if( p )
	{
           /* place address to write word in data [0]; */
           data[0] = (unsigned long) p;
           /* place value to write @ address data [1]; */
           data[1] = (unsigned long) SwapLong( v );
           /* pram3 = function: 1=write8bits,2=write16bits,4=write32bits */
           write( pAP->nAPDeviceHandle, &data[0], 4 );
	}
}


long get_param()
{

    int temp;

    printf("enter hex parameter: ");
    scanf("%x",&temp);
    printf("\n");
    return((long)temp);
}

#endif /* NO_AP_IOFUNCTIONS */


/*
 Blocking options

 parameter =  0 = byte write & block
 parameter =  1 = word write & block
 parameter =  2 = 32 bit write & block
 parameter = 10 = no write, just block & wait for an input event to wake up

 Returns the interrupt pending status value.
*/

uint32_t APBlockingStartConvert(int nHandle, long *p, long v, long parameter)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data[4];

	pAP = GetAP(nHandle);
	if(pAP == NULL)
	   return(0);

	data[0] = (unsigned long) p;		/* place address of write in data[0] */

	switch(parameter)
	{
		case 0:	/* flag=0=byte write */
		    data[1] = (unsigned long) v;		/* place byte value to write @ address data[1]; */
		break;
		case 1:	/* flag=1=word write */
		    data[1] = (unsigned long) SwapBytes( v );	/* place word value to write @ address data[1]; */
		break;
		case 2:	/* flag=2=long write */
		    data[1] = (unsigned long) SwapLong( v );	/* place long value to write @ address data[1]; */
		break;
	}

 	data[2] = (unsigned long)parameter;	/* save Blocking option parameter */

 	/* place board instance index in data[3] */
 	data[3] = (unsigned long) pAP->nDevInstance;	/* Device Instance */

 	write( pAP->nAPDeviceHandle, &data[0], 8 );		/* function: 8=blocking_start_convert */

	return( (uint32_t)SwapLong( (long)data[1] ) );	/* return Interrupt Pending value */
}


void APTerminateBlockedStart(int nHandle)
{
	APDATA_STRUCT* pAP;	/*  local */
	unsigned long data;


	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return;

	data = (unsigned long) pAP->nDevInstance;	/* Device Instance */
	ioctl( pAP->nAPDeviceHandle, 21, &data );	/* get wake up/terminate cmd */
}


APSTATUS GetAPAddress(int nHandle, long* pAddress)
{
	APDATA_STRUCT* pAP;	/*  local */

	pAP = GetAP(nHandle);
	if(pAP == NULL)
		return E_INVALID_HANDLE;

	*pAddress = pAP->lBaseAddress;
	return (APSTATUS)S_OK;
}


APSTATUS EnableAPInterrupts(int nHandle)
{
	APDATA_STRUCT* pAP;
	AP_BOARD_MEMORY_MAP* pAPCard;
	word nValue;	/* new */

	pAP = GetAP(nHandle);
	if(pAP == 0)
		return E_INVALID_HANDLE;

	if(pAP->bInitialized == FALSE)
		return E_NOT_INITIALIZED;

	/* Enable interrupts */
	pAPCard = (AP_BOARD_MEMORY_MAP*)pAP->lBaseAddress;
	nValue = (word)input_long( nHandle,(long*)&pAPCard->InterruptRegister);
	output_long( nHandle, (long*)&pAPCard->InterruptRegister, (long)( nValue | AP_INT_ENABLE ));
	pAP->bIntEnabled = TRUE;	/* mark interrupts enabled */
	return (APSTATUS)S_OK;
}


APSTATUS DisableAPInterrupts(int nHandle)
{
	APDATA_STRUCT* pAP;
	AP_BOARD_MEMORY_MAP* pAPCard;
	word nValue;	/* new */

	pAP = GetAP(nHandle);
	if(pAP == 0)
		return E_INVALID_HANDLE;

	if(pAP->bInitialized == FALSE)
		return E_NOT_INITIALIZED;

	/* Disable interrupt */
	pAPCard = (AP_BOARD_MEMORY_MAP*)pAP->lBaseAddress;
	nValue = (word)input_long( nHandle,(long*)&pAPCard->InterruptRegister);
	nValue &= ~AP_INT_ENABLE;
	output_long( nHandle, (long*)&pAPCard->InterruptRegister, (long)( nValue ));
	pAP->bIntEnabled = FALSE;	/* mark interrupts disabled */
	return (APSTATUS)S_OK;
}


APSTATUS InitAPLib(void)
{
	int i;				/* General purpose index */

        if( gNumberAPs == -1)		/* first time used - initialize pointers to 0 */
        {
	  gNumberAPs = 0;		/* Initialize number of APs to 0 */

	  /* initialize the pointers to the AP data structure */
	  for(i = 0; i < MAX_APS; i++)
		gpAP[i] = 0;		/* set to a NULL pointer */
        }
	return (APSTATUS)S_OK;
}


APSTATUS APOpen(int nDevInstance, int* pHandle, char* devname)
{
	APDATA_STRUCT* pAP;		/* local pointer */
	unsigned long data[MAX_APS];
	char devnamebuf[64];
	char devnumbuf[8];

	*pHandle = -1;		/* set callers handle to an invalid value */

	if(gNumberAPs == MAX_APS)
		return E_OUT_OF_APS;

	/* Allocate memory for a new AP structure */
	pAP = (APDATA_STRUCT*)malloc(sizeof(APDATA_STRUCT));

	if(pAP == 0)
		return (APSTATUS)E_OUT_OF_MEMORY;

	pAP->nHandle = -1;
	pAP->bInitialized = FALSE;
	pAP->bIntEnabled = FALSE;
	pAP->nAPDeviceHandle = 0;
	pAP->lBaseAddress = 0;
	pAP->nInteruptID = 0;
	pAP->nIntLevel = 0;
	pAP->nDevInstance = -1;	/* Device Instance */

	memset( &pAP->devname[0], 0, sizeof(pAP->devname));
	memset( &devnamebuf[0], 0, sizeof(devnamebuf));
	memset( &devnumbuf[0], 0, sizeof(devnumbuf));

	strcpy(devnamebuf, "/dev/");
	strcat(devnamebuf, devname);
	sprintf(&devnumbuf[0],"%d",nDevInstance);
	strcat(devnamebuf, devnumbuf);

	pAP->nAPDeviceHandle = open( devnamebuf, O_RDWR );

	if( pAP->nAPDeviceHandle < 0 )
	{
        	free((void*)pAP);		/* delete the memory for this AP */
			return (APSTATUS)ERROR;
	}
	strcpy(&pAP->devname[0], &devnamebuf[0]);	/* save device name */
	pAP->nDevInstance = nDevInstance;	/* Device Instance */

	/* Get Base Address */
	memset( &data[0], 0, sizeof(data)); /* no mem if data[x] returns 0 from ioctl() */
	ioctl( pAP->nAPDeviceHandle, 5, &data[0] );		/* get address cmd */
	pAP->lBaseAddress = data[nDevInstance];

	/* Get IRQ Number */
	ioctl( pAP->nAPDeviceHandle, 6, &data[0] );		/* get IRQ cmd */
	pAP->nIntLevel = ( int )( data[nDevInstance] & 0xFF );

	AddAP(pAP);                  /* call function to add AP to array and set handle */
	*pHandle = pAP->nHandle;      /* return our handle */

	return (APSTATUS)S_OK;
}


APSTATUS APClose(int nHandle)
{
	/*  Delete the AP with the provided handle */
	APDATA_STRUCT* pAP;	/* local pointer */

	pAP = GetAP(nHandle);

	if(pAP == 0)
		return E_INVALID_HANDLE;

	if(pAP->bInitialized == FALSE)
		return E_NOT_INITIALIZED;

  	close( pAP->nAPDeviceHandle );

  	pAP->nAPDeviceHandle = -1;
	DeleteAP(nHandle);		/*  Delete the AP with the provided handle */

	return (APSTATUS)S_OK;
}


APSTATUS APInitialize(int nHandle)
{
	APDATA_STRUCT* pAP;

	pAP = GetAP(nHandle);
	if(pAP == 0)
		return E_INVALID_HANDLE;

	pAP->bInitialized = TRUE;	/* AP is now initialized */

	return (APSTATUS)S_OK;
}


void AddAP(APDATA_STRUCT* pAP)
{
	int i, j;			/* general purpose index */
	BOOL bFound;			/* general purpose BOOL */

	for(i = 0; i < MAX_APS; i++)	/* Determine a handle for this AP */
	{
		bFound = TRUE;
		for(j = 0; j < gNumberAPs; j++)
		{
			if(i == gpAP[j]->nHandle)
			{
				bFound = FALSE;
				break;
			}
		}

		if(bFound)
			break;
	}

	pAP->nHandle = i;          	/* set new handle */
	gpAP[gNumberAPs] = pAP;		/* add AP to array */
	gNumberAPs++;			/* increment number of APs */
}


void DeleteAP(int nHandle)
{
	APDATA_STRUCT* pAP;
	int i;

	if(gNumberAPs == 0)
		return;

	pAP = 0;			/* initialize pointer to null */
	for(i = 0; i < gNumberAPs; i++)/* Find AP that has this handle */
	{
		if(nHandle == gpAP[i]->nHandle)
		{
			pAP = gpAP[i];
			break;
		}
	}
	if(pAP == 0)			/* return if no AP has been found */
		return;

	free((void*)pAP);		/* delete the memory for this AP */

	/* Rearrange AP array */
	gpAP[i] = gpAP[gNumberAPs - 1];
	gpAP[gNumberAPs - 1] = 0;
	gNumberAPs--;			/* decrement AP count */
}


APDATA_STRUCT* GetAP(int nHandle)
{
	APDATA_STRUCT* pAP;
	int i;				/* General purpose index */

	for(i = 0; i < gNumberAPs; i++)/* Find AP that has this handle */
	{
		if(nHandle == gpAP[i]->nHandle)
		{
			pAP = gpAP[i];
			return pAP;
		}
	}
	return (APDATA_STRUCT*)0;	/* return null */
}
