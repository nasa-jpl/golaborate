#include <math.h>
#include "../apcommon/apcommon.h"
#include "AP236.h"

/*
{+D}
    SYSTEM:	    Library Software

    FILENAME:	    cd236.c

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:	    This module is used to correct output conversions for the board.

    CALLING
    SEQUENCE:       void cd236(struct cblk236 *c_blk, int channel, double Volts);
			where:
			        ptr (pointer to structure)
			            Pointer to the configuration block structure.
				int channel
				    Channel to correct
				double double Volts
				    Volt value to correct

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

    This module is used to correct the input data for the board.
    A pointer to the Configuration Block will be passed to
    this routine.  The routine will use a pointer within the Configuration
    Block to reference the registers on the Board.
*/


void cd236(struct cblk236 *c_blk, int channel, double Volts)
{

/*
    declare local storage
*/

    double f_cor;
    int range;

/*
        Entry point of routine
	Storage configuration for offset & gain correction pairs[2] for each range[8] for each channel[16]
*/

    range = (int)(c_blk->opts.chan[channel].Range & 0x7);	/* get channels range setting */

    f_cor = ((1.0 + (double)c_blk->ogc236[channel][range][GAIN] / 1048576.0) * (*c_blk->pIdealCode)[range][IDEALSLOPE]) *
		Volts + (*c_blk->pIdealCode)[range][IDEALZEROBTC] + ((double)c_blk->ogc236[channel][range][OFFSET] / 16.0);

    f_cor += (f_cor < 0.0) ? -0.5 : 0.5; /* round */

    f_cor = fmin(f_cor, (*c_blk->pIdealCode)[range][CLIPHI]);
    f_cor = fmax(f_cor, (*c_blk->pIdealCode)[range][CLIPLO]);
/*
printf("Ch %X R = %X IZ %lf IS %lf Oc = %lf Gc = %lf fc = %lf\n", channel, range, (*c_blk->pIdealCode)[range][IDEALZEROBTC],
(*c_blk->pIdealCode)[range][IDEALSLOPE], (double)c_blk->ogc236[channel][range][OFFSET], (double)c_blk->ogc236[channel][range][GAIN], f_cor);
*/
    c_blk->cor_buf[channel] = (short)f_cor;
}
