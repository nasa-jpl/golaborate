
#include "apcommon.h"
#include "AP236.h"

/*
{+D}
    SYSTEM:	    Library Software

    FILENAME:	    wro236.c

    MODULE NAME:    wro236 - write analog output for the board.

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:      This module is used to perform the write output function
                   for the board.

    CALLING
    SEQUENCE:   wro236(channel,data);
         where:
         channel (unsigned short)
                Value of the analog output channel number (0-7).
         data (unsigned short)
               Value of the analog output data.

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

    This module is used to perform the write output function
    for the board.  A structure pointer to the board memory map,
    the analog output channel number value, and the analog output
    data value will be passed to this routine.  The routine writes
    the output data to the analog output channel register on the board.
*/


void wro236(struct cblk236 *c_blk, int channel, word data)
{

/*
         Declare local data areas
*/

uint32_t wdata;

/*
    ENTRY POINT OF ROUTINE:
    Write the output data to the output channel on the board.
*/

  data ^= 0x8000; /* Convert BTC data to straight binary data */

  if( c_blk->opts.chan[channel].UpdateMode ) /* 1 = simultaneous mode */
        wdata = (SMWrite << 16);
  else
        wdata = (TMWrite << 16);

  wdata |= data;	/* append data to the shift register update command */

/*printf("Wro236 %X %X\n", data, wdata);*/

  output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->dac_reg[channel], wdata );
  usleep((useconds_t) 2);	/* write delay */
}


/*
{+D}
    SYSTEM:	    Library Software

    FILENAME:	    wro236.c

    MODULE NAME:    simtrig236 - Simultaneous Trigger for the board.

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:	    This module is used to Simultaneous Trigger conversions for the board.

    CALLING
    SEQUENCE:	    void simtrig236(struct cblk236 *c_blk);

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

    This module is used to start conversions on the board.
*/
void simtrig236(struct cblk236 *c_blk)
{

/*
    ENTRY POINT OF ROUTINE:

    Write the output data to the output channel on the board
*/
    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->SimultaneousOutputTrigger, 1 );
}

