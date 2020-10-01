
#include "apcommon.h"
#include "AP235.h"

/*
{+D}
    SYSTEM:	    Library Software

    FILENAME:	    wro235.c

    MODULE NAME:    fifowro235 - write analog output to FIFO.

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:	    This module is used to perform a FIFO or single value
                    write output function for the selected channel.
    CALLING
	SEQUENCE:   fifowro235(c_blk,channel);
		    where:
			c_blk (prt)
			    pointer to configuration structure.
			channel (int)
			    Channel to write to.

    MODULE TYPE:    int

    I/O RESOURCES:

    SYSTEM
	RESOURCES:

    MODULES
	CALLED:

    REVISIONS:

  DATE	     BY	    PURPOSE
  --------  ----    ------------------------------------------------

{-D}
*/


/*
    MODULES FUNCTIONAL DETAILS:

    This module is used to perform a FIFO write output function for the
    selected channel.  A structure pointer to the board memory map, the analog
    output channel number value will be passed to this routine.  The routine
    writes the output data to the FIFO on the board.
*/


void fifowro235(struct cblk235 *c_blk, int channel)
{

/*
    Declare local data areas
*/

uint32_t wdata, cnt;

/*
    ENTRY POINT OF ROUTINE:
    Write the output data to the output channel on the board.
*/

   if( c_blk->opts.chan[channel].OpMode == DAC_FIFO )	/* DAC_FIFO conversion mode ? */
   {
     /* write half the number of samples from the buffer into the channel FIFO */
     /* pack two 16 bit samples into a 32 bit value to increase data throughput when writing to the DAC */
     for(cnt = (c_blk->SampleCount[channel] >> 2); cnt; cnt--)
     {
        wdata = (uint32_t)(*c_blk->current_ptr[channel]++ & 0xFFFF);	/* sample lo */

        if(c_blk->current_ptr[channel] >= c_blk->tail_ptr[channel])
           c_blk->current_ptr[channel] = c_blk->head_ptr[channel];

        wdata |= (uint32_t)*c_blk->current_ptr[channel]++ << 16;	/* sample hi */

        if(c_blk->current_ptr[channel] >= c_blk->tail_ptr[channel])
           c_blk->current_ptr[channel] = c_blk->head_ptr[channel];

        output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->DAC[channel].Fifo, (long)wdata);
     }
   }
   else		/* Single value DAC Direct Access */
   {
      if( c_blk->opts.chan[channel].UpdateMode ) /* 1 = simultaneous mode */
        wdata = (SMWrite << 16);
      else
        wdata = (TMWrite << 16);

      wdata |= (word)*c_blk->head_ptr[channel]; /* append data to the shift register update command */

      output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->DAC[channel].DirectAccess, wdata );

      usleep((useconds_t) 2);	/* write delay */
   }
}



/*
{+D}
    SYSTEM:	    Library Software

    FILENAME:	    wro235.c

    MODULE NAME:    fifodmawro235 - DMA analog output to FIFO.

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:	    This module is used to perform a FIFO DMA
                    write output function for the selected channel.
    CALLING
	SEQUENCE:   fifodmawro235(c_blk,channel);
		    where:
			c_blk (prt)
			    pointer to configuration structure.
			channel (int)
			    Channel to write to.

    MODULE TYPE:    int

    I/O RESOURCES:

    SYSTEM
	RESOURCES:

    MODULES
	CALLED:

    REVISIONS:

  DATE	     BY	    PURPOSE
  --------  ----    ------------------------------------------------

{-D}
*/


/*
    MODULES FUNCTIONAL DETAILS:

    This module is used to perform a DMA FIFO write output function for the
    selected channel.  A structure pointer to the board memory map, the analog
    output channel number value will be passed to this routine.  The routine
    writes the output data to the FIFO on the board.
*/


void fifodmawro235(struct cblk235 *c_blk, int channel)
{

/*
    Declare local data areas
*/

    int i;
    uint32_t lValue;
    struct mapap235* pAPCard;/* board pointer */
    /* internal & external address pointers to the transfer descriptor list used by scatter-gather DMA */
    struct scatterAP235list *IxSGLPtr, *ExSGLPtr;
    static unsigned int pingpong[16] = { 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0 }; /* pingpong buffer toggle flags */

/*
    ENTRY POINT OF ROUTINE:
*/

    /*          8 KByte RAM On Board Memory            */
    /*            Internal Address 0xA000              */
    /*   --------------------------------------------  */
    /*  |                8 KByte RAM                 | */
    /*  |                                            | */
    /*  |    Room for 128, 64 byte Scatter-gather    | */
    /*  |                Descriptors                 | */
    /*   --------------------------------------------  */
    /* On board Scatter-gather list for DMA from system memory to board channel FIFO registers */

    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->CDMAControlRegister, DMAReset );	/* Reset device */

    pAPCard = (struct mapap235*)NULL;	/* force address to be zero for board internal address pointer */

    if(pingpong[channel]) /* based on the pingpong flag set up the first or second page scatter list address pointers */
    {
      /* second page scatter list external address pointer */
      ExSGLPtr = (struct scatterAP235list *)((byte*)&c_blk->brd_ptr->CHAN[channel].sptrlo.NxtDescPtrLo);
      /* second page scatter list internal address pointer */
      IxSGLPtr = (struct scatterAP235list *)((byte*)&pAPCard->CHAN[channel].sptrlo.NxtDescPtrLo);
    }
    else
    {
      /* first page scatter list external address pointer */
      ExSGLPtr = (struct scatterAP235list *)((byte*)&c_blk->brd_ptr->CHAN[channel].fptrlo.NxtDescPtrLo);
      /* first page scatter list internal address pointer */
      IxSGLPtr = (struct scatterAP235list *)((byte*)&pAPCard->CHAN[channel].fptrlo.NxtDescPtrLo);
    }

    /* initialize (zero) status member of the DMA Descriptor list for translation reg addr Lo */
    output_long( c_blk->nHandle, (long*)&ExSGLPtr[0].Status, 0);

    /* initialize (zero) status member of the DMA Descriptor list for translation reg addr Hi */
    output_long( c_blk->nHandle, (long*)&ExSGLPtr[1].Status, 0);

    /* initialize (zero) status member of the DMA Descriptor list for the page data transfer */
    output_long( c_blk->nHandle, (long*)&ExSGLPtr[2].Status, 0);

    /* Verify device idle */
    lValue = input_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->CDMAStatusRegister );
    if( lValue & DMATransferComplete)
    {
      /* set control register, scatter-gather DMA mode and DMA Key Hole Write */
      output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->CDMAControlRegister, 0x2A);

      /* CDMA Descriptor Pointer Register the internal address of the scatter/gather list start address */
      output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->CDMADescriptorPointerRegister, (long)IxSGLPtr );

      /* CDMA Tail Descriptor Pointer Register the internal address of the scatter/gather list end address - the write also starts the DMA transfer */
      output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->CDMATailDescriptorPointerRegister, (long)((byte *)IxSGLPtr) + 0x80);

      pingpong[channel] ^= 1; /* update the pingpong buffer toggle flag - set/clear as needed */

      for(i = 0; i < DMAMAX_TRIES; i++ ) /* wait for DMA transfer to complete (device idle) or timeout */
      {
         usleep(20);
         lValue = input_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->CDMAStatusRegister );
         if( lValue & DMATransferComplete )/* Transfer Complete? */
             break;
      }

      if( i >= DMAMAX_TRIES )
          printf("\nDMA timeout!\n");
     }
     else
          printf("\nDevice not idle status = %X\n",lValue);
}



/*
{+D}
    SYSTEM:	    Library Software

    FILENAME:	    wro235.c

    MODULE NAME:    simtrig235 - Simultaneous Trigger for the board.

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:	    This module is used to Simultaneous Trigger conversions for the board.

    CALLING
    SEQUENCE:	    void simtrig235(struct cblk235 *c_blk);

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
void simtrig235(struct cblk235 *c_blk)
{

/*
    ENTRY POINT OF ROUTINE:

    Write the output data to the output channel on the board
*/

    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->SoftwareTrigger, 1 );
}

