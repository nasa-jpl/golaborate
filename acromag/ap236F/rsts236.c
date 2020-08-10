
#include "../apcommon/apcommon.h"
#include "AP236.h"

/*
{+D}
    SYSTEM:	    AP236 Board

    FILENAME:	    rsts236.c

    MODULE NAME:    rsts236 - read status of board

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    FJM

    CODED BY:	    FJM

    ABSTRACT:	    This module is used to read status of the board.

    CALLING
	SEQUENCE:   rsts236(ptr);
		    where:
			ptr (pointer to structure)
			    Pointer to the status block structure.

    MODULE TYPE:    void

    I/O RESOURCES:

    SYSTEM
	RESOURCES:

    MODULES
	CALLED:

    REVISIONS:

  DATE	  BY	    PURPOSE
-------  ----	------------------------------------------------

{-D}
*/


/*
    MODULES FUNCTIONAL DETAILS:

    This module is used to perform the read status function for the
    board.  A pointer to the Status Block will be passed to this routine.
    The routine will use a pointer within the Status Block together with
    offsets to reference the registers on the Board and will transfer the
    status information from the Board to the Status Block.
*/


void rsts236(c_blk)
struct cblk236 *c_blk;
{

   long addr, index;

/*
    ENTRY POINT OF ROUTINE
*/

   c_blk->revision = input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->FirmwareRevision ); /* read board Revision */

   /* read temp & VCC info from FPGA */
   for( addr = 0, index = 0; index < 3; index++, addr++)
   {
     output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->XDAC_AddressReg, (long)addr); /* FPGA[addr] */
     c_blk->FPGAAdrData[index] = input_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->XDAC_StatusControl); /* addr & data [index] */
   }

   for( addr = 0x20, index = 3; index < 6; index++, addr++)
   {
     output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->XDAC_AddressReg, (long)addr); /* FPGA[addr] */
     c_blk->FPGAAdrData[index] = input_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->XDAC_StatusControl); /* addr & data [index] */
   }

   for( addr = 0x24, index = 6; index < 9; index++, addr++)
   {
     output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->XDAC_AddressReg, (long)addr); /* FPGA[addr] */
     c_blk->FPGAAdrData[index] = input_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->XDAC_StatusControl); /* addr & data [index] */
   }
}

