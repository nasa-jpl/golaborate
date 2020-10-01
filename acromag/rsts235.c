
#include "apcommon.h"
#include "AP235.h"

/*
{+D}
    SYSTEM:         Library Software - AP235 Board

    MODULE NAME:    rstsAP235 - read status of AP235 board

    VERSION:        A

    CREATION DATE:  12/01/15

    DESIGNED BY:    FM

    CODED BY:       FM

    ABSTRACT:       This module is used to read status of the AP235 board.

    CALLING
        SEQUENCE:   rstsAP235(ptr);
                    where:
                        ptr (pointer to structure)
                            Pointer to the configuration block structure.

    MODULE TYPE:    void

    I/O RESOURCES:

    SYSTEM
        RESOURCES:

    MODULES
        CALLED:

    REVISIONS:

  DATE    BY        PURPOSE
-------  ----   ------------------------------------------------

{-D}
*/


/*
    MODULES FUNCTIONAL DETAILS:

    This module is used to perform the read status function
    for the AP235 board.  A pointer to the Configuration Block will
    be passed to this routine.  The routine will use a pointer
    within the Configuration Block together with offsets
    to reference the registers on the Board and will transfer the
    status information from the Board to the Configuration Block.
*/



void rsts235(c_blk)
struct cblk235 *c_blk;
{

int i;

/*
    ENTRY POINT OF ROUTINE:
    read board information
*/

   c_blk->location = (word)input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->LocationRegister);/* AP location */

   c_blk->revision = input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->FirmwareRevision);	/* AP Revision */

   for(i = 0; i < 16; i++)
     c_blk->ChStatus[i] = input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->DAC[i].Status);	/* DAC channel status */

   /* read temp & VCC info from FPGA into status structure */
   /* temperature Data Register | (MS 16 bits addr 200) */

   c_blk->FPGAAdrData[0] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_Temperature) | 0x2000000);

   /* supply monitor Data Register | (MS 16 bits addr 204) */
   c_blk->FPGAAdrData[1] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_VCCInt) | 0x2040000);

   /* supply monitor Data Register | (MS 16 bits addr 208) */
   c_blk->FPGAAdrData[2] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_VCCAux) | 0x2080000);

   /* MAXtemperature Data Register | (MS 16 bits addr 280) */
   c_blk->FPGAAdrData[3] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_MAXTemperature) | 0x2800000);

   /* MAXsupply monitor Data Register | (MS 16 bits addr 284) */
   c_blk->FPGAAdrData[4] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_MAXVCCInt) | 0x2840000);

   /* MAXsupply monitor Data Register | (MS 16 bits addr 288) */
   c_blk->FPGAAdrData[5] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_MAXVCCAux) | 0x2880000);

   /* MINtemperature Data Register | (MS 16 bits addr 290) */
   c_blk->FPGAAdrData[6] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_MINTemperature) | 0x2900000);

   /* MINsupply monitor Data Register (MS 16 bits addr 294) */
   c_blk->FPGAAdrData[7] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_MINVCCInt) | 0x2940000);

   /* MINsupply monitor Data Register (MS 16 bits addr 298) */
   c_blk->FPGAAdrData[8] = (input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->XW_MINVCCAux) | 0x2980000);
}

