#include <math.h>
#include "../apcommon/apcommon.h"
#include "AP236.h"


/*
{+D}
    SYSTEM:         Software

    FILENAME:       drvr236.c

    MODULE NAME:    main - main routine of example software.

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:       This module is the main routine for the example program
                    which demonstrates how the Library is used.

    CALLING
        SEQUENCE:   

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

        This module is the main routine for the example program
        which demonstrates how the Library is used.
*/


/* Ideal Zero SB, BTC, Slope, endpoint, and clip constants
   ranges[8], ideal straight binary is [0], ideal 2'Comp is [1], slope is [2], 
   endpoint low is [3], endpoint high is [4], clip low is [5] clip high is [6]
*/

double IdealCode[8][7] =
{
  /* IdealZeroSB, IdealZeroBTC, IdealSlope, -10 to 10V, cliplo, cliphi */
  { 32768.0, 0.0, 3276.8, -10.0, 10.0, -32768.0, 32767.0},

  /* IdealZeroSB, IdealZeroBTC, IdealSlope,   0 to 10V, cliplo, cliphi */
  { 0.0, -32768.0, 6553.6, 0.0, 10.0, -32768.0, 32767.0},

  /* IdealZeroSB, IdealZeroBTC, IdealSlope,  -5 to  5V, cliplo, cliphi */
  { 32768.0, 0.0, 6553.6, -5.0, 5.0, -32768.0, 32767.0},

  /* IdealZeroSB, IdealZeroBTC, IdealSlope,   0 to  5V, cliplo, cliphi */
  { 0.0, -32768.0, 13107.2, 0.0, 5.0, -32768.0, 32767.0},

  /* IdealZeroSB, IdealZeroBTC, IdealSlope, -2.5 to 7.5V, cliplo, cliphi */
  { 16384.0,-16384.0, 6553.6, -2.5, 7.5, -32768.0, 32767.0},

  /* IdealZeroSB, IdealZeroBTC, IdealSlope,  -3 to  3V, cliplo, cliphi */
  { 32768.0 ,0.0, 10922.67, -3.0, 3.0, -32768.0, 32767.0},

  /* IdealZeroSB, IdealZeroBTC, IdealSlope, 0V to +16V, cliplo, cliphi */
  { 0.0, -32768.0, 4095.9, 0.0, 16.0, -32768.0, 32767.0 },

  /* IdealZeroSB, IdealZeroBTC, IdealSlope, 0V to +20V, cliplo, cliphi */
  { 0.0, -32768.0, 3276.8, 0.0, 20.0, -32768.0, 32767.0 },
};



int main(argc, argv)
int argc; char *argv[];
{


/*
    DECLARE LOCAL DATA AREAS:
*/

    char cmd_buff[32];			/* command line input buffer */
    unsigned finished, finished2;	/* flag to exit program */
    long addr;				/* holds board address */
    int item;				/* menu item selection variable */
    int i;				/* loop index */
    int range;
    int temp;
    double Volts, ideal;
    double zero, span, slope, gcoef, ocoef; /* storage for coefficient calibration */
    struct cblk236 c_block236;		/* configuration block */
    int ap_instance = 0;
    int current_channel = 0;

/*
    ENTRY POINT OF ROUTINE:
    INITIALIZATION
*/

    if(argc == 2)
      ap_instance = atoi(argv[1]);

    finished2 = finished = 0;	/* indicate not finished */
    range = 0;
    memset(&c_block236, 0, sizeof(c_block236));

    /* Put the address of the initialized constant array into the configuration block structure */
    c_block236.pIdealCode = &IdealCode;	/* pointer to Ideal Zero straight binary, and Slope constants */

/*
    Initialize the Configuration Parameter Block to default values.
*/

    c_block236.bAP = FALSE;		/* indicate not initialized and set up yet */
    c_block236.bInitialized = FALSE;	/* indicate not ready */
    c_block236.nHandle = 0;		/* make handle to a closed board */

/*
	Initialize the AP library
*/
    if(InitAPLib() != S_OK)
    {
	printf("\nUnable to initialize the AP library. Exiting program.\n");
	exit(0);
    }

/*
	Open an instance of a AP device
	Other device instances can be obtained
	by changing parameter 1 of APOpen()
*/
    if(APOpen(ap_instance, &c_block236.nHandle, DEVICE_NAME ) != S_OK)
    {
	  printf("\nUnable to Open instance of AP236.\n");
	  finished = 1;	 /* indicate finished with program */
    }
    else
    {
      if(APInitialize(c_block236.nHandle) == S_OK)/* Initialize */
      {
	GetAPAddress(c_block236.nHandle, &addr);	/* Read back address */
	c_block236.brd_ptr = (struct map236 *)addr;
	c_block236.bInitialized = TRUE;
	c_block236.bAP = TRUE;
      }
    }

/*
    Enter main loop
*/      

    while(!finished)
    {
      range = (int)(c_block236.opts.chan[current_channel].Range & 0x7); /* get current channels range setting */
      printf("\nAP236 Library Demonstration  Rev. A");
      printf("   Channel: %X, Range:%7.3lf to%7.3lf\n", current_channel,
		 (*c_block236.pIdealCode)[range][ENDPOINTLO], (*c_block236.pIdealCode)[range][ENDPOINTHI]);
      printf(" 1. Exit this Program\n");
      printf(" 2. Read Calibration Coefficients\n");
      printf(" 3. Read Status Command\n");
      printf(" 4. Examine/Change Current Channel\n");
      printf(" 5. Set Up Configuration Block Parameters\n");
      printf(" 6. Configure Current Channel\n");
      printf(" 7. Write Ideal Data To Output\n");
      printf(" 8. Write Corrected Data To Output\n");
      printf(" 9. Simultaneous Trigger\n");
      printf("10. Display Ideal/Corrected Data, Offset/Gain Coefficients\n");
      printf("11. Clear Data Buffers\n");
      printf("12. Alter Offset/Gain Coefficients\n");
      printf("Select: ");
      scanf("%d",&item);

/*
    perform the menu item selected.
*/  

      switch(item)
      {
        case 1: /* exit program command */
            printf("Exit program(y/n)?: ");
            scanf("%s",cmd_buff);
            if( cmd_buff[0] == 'y' || cmd_buff[0] == 'Y' )
                finished++;
        break;

        case 2: /* Read Calibration Coefficients */
              memset(&c_block236.IDbuf[0],0,sizeof(c_block236.IDbuf));	/* empty the buffer */
              ReadFlashID236(&c_block236, &c_block236.IDbuf[0]);

              if( (strstr( (const char *)&c_block236.IDbuf[0], (const char *)FlashIDString ) == NULL) )	/* AP220 or AP236 ID */
		  printf("\nUnable to read APBoard FLASH ID.\n");
              else
              {
		printf("\n%s found, reading calibration coefficients...\n",&c_block236.IDbuf[0]);
		rcc236(&c_block236); /* read the calibration coef. into an array */
	      }
        break;

	case 3:     /* read board status */

	    if(!c_block236.bInitialized)
		printf("\n>>> ERROR: BOARD NOT SET UP <<<\n");
	    else
		psts236(&c_block236);
	break;

        case 4:	/* Setup current channel */
	    selectch236(&current_channel);
        break;

	case 5: /* set up configuration block parameters */
	    scfg236(&c_block236, current_channel);
	break;

	case 6: /* configure board command */
	    if(!c_block236.bInitialized)
		printf("\n>>> ERROR: BOARD NOT SET UP <<<\n");
	    else
		cnfg236(&c_block236, current_channel); /* configure channel */
	break;

        case 7: /* Write Ideal Data To Output */
            if(!c_block236.bInitialized)
		printf("\n>>> ERROR: BOARD NOT SET UP <<<\n");
            else
            {
		/* get current channels range setting */
		range = (int)(c_block236.opts.chan[current_channel].Range & 0x7);

                while(1){
                    printf("Enter desired voltage for channel %X: ie: 1.25    ",current_channel);
                    scanf("%lf", &Volts);
                    if(Volts >= (*c_block236.pIdealCode)[range][ENDPOINTLO] &&
				 Volts <= (*c_block236.pIdealCode)[range][ENDPOINTHI])
                       break;
                    printf("\n>>> Voltage out of range <<<\n");
                }

		ideal = (((*c_block236.pIdealCode)[range][IDEALSLOPE] * Volts) + (*c_block236.pIdealCode)[range][IDEALZEROBTC]);
		ideal += (ideal < 0.0) ? -0.5 : 0.5; /* round */
		ideal = fmin(ideal, (*c_block236.pIdealCode)[range][CLIPHI]);
		ideal = fmax(ideal, (*c_block236.pIdealCode)[range][CLIPLO]);
                c_block236.ideal_buf[current_channel] = (word)ideal;

                wro236( &c_block236, current_channel,(word)(c_block236.ideal_buf[current_channel]));
            }
        break;
        case 8: /* Write Corrected Data To Output */
            if(!c_block236.bInitialized)
		printf("\n>>> ERROR: BOARD NOT SET UP <<<\n");
            else
            {
		/* get current channels range setting */
		range = (int)(c_block236.opts.chan[current_channel].Range & 0x7);

                while(1){
                    printf("Enter desired voltage for channel %X: ie: 1.25    ",current_channel);
                    scanf("%lf", &Volts);
                    if(Volts >= (*c_block236.pIdealCode)[range][ENDPOINTLO] &&
				 Volts <= (*c_block236.pIdealCode)[range][ENDPOINTHI])
                       break;
                    printf("\n>>> Voltage out of range <<<\n");
                }

                cd236(&c_block236, current_channel, Volts);	/* correct data for channel */ 

                wro236( &c_block236, current_channel,(word)(c_block236.cor_buf[current_channel]));
            }
        break;

        case 9:  /* Simultaneous Trigger */
            if(!c_block236.bInitialized)
		printf("\n>>> ERROR: BOARD NOT SET UP <<<\n");
            else
                simtrig236(&c_block236);	/* Simultaneous Trigger */
        break;

        case 10: /* Display Ideal/Corrected Data, Offset/Gain Coefficients */

            printf("\n  ");
            for(i = 0; i < 8; i++)
                printf("    CH %02d",i);

            printf("\nID");
            for(i = 0; i < 8; i++)
                printf("     %04X",(word)c_block236.ideal_buf[i]);

            printf("\nCD");
            for(i = 0; i < 8; i++)
                printf("     %04X",(word)c_block236.cor_buf[i]);

            printf("\nGD");
            for(i = 0; i < 8; i++)
            {
		/* get current channels range setting */
		range = (int)(c_block236.opts.chan[i].Range & 0x7);
		printf("     %04X",(c_block236.ogc236[i][range][GAIN] & 0xFFFF));
            }

            printf("\nOD");
            for(i = 0; i  < 8; i++)
            {
		/* get current channels range setting */
		range = (int)(c_block236.opts.chan[i].Range & 0x7);
		printf("     %04X",(c_block236.ogc236[i][range][OFFSET] & 0xFFFF));
            }

            printf("\n");
        break;

        case 11:  /* clear data buffers */
              memset(&c_block236.cor_buf[0],0,sizeof(c_block236.cor_buf));	/* empty the buffer */
              memset(&c_block236.ideal_buf[0],0,sizeof(c_block236.ideal_buf));	/* empty the buffer */
        break;

        case 12:   /* Gain/Offset Coefficients */
            if(!c_block236.bInitialized)
		printf("\n>>> ERROR: BOARD NOT SET UP <<<\n");
            else
	    {
	      finished2 = 0;
	      printf("\n\nNote: It is recommended that the factory calibration values\n");
	      printf("      not be changed unless you are familiar with making\n");
	      printf("      these types of measurements and use a voltmeter capable\n");
	      printf("      of 16-bit resolution and accuracy.\n");

	      while(!finished2)
	      {
		/* get current channels range setting */
		range = (int)(c_block236.opts.chan[current_channel].Range & 0x7);
		cnfg236(&c_block236, current_channel); /* configure channel... make sure settings are in place */
	        printf("\nAlter Gain/Offset Coefficients\n");
	        printf("\nCurrent Channel Number:     %X",current_channel);
		printf("\nRange:%7.3lf to%7.3lf",
			(*c_block236.pIdealCode)[range][ENDPOINTLO], (*c_block236.pIdealCode)[range][ENDPOINTHI]);
	        printf("\nCurrent Gain Coefficient:   %04X", (word)c_block236.ogc236[current_channel][range][GAIN]);
	        printf("\nCurrent Offset Coefficient: %04X\n\n", (word)c_block236.ogc236[current_channel][range][OFFSET]);

	        printf("1. Return to Previous Menu\n");
		printf("2. Read Flash Calibration Coefficients\n");
		printf("3. Change Gain Coefficient\n");
		printf("4. Change Offset Coefficient\n");
		printf("5. Change Channel Number\n");
		printf("6. Calculate New Offset/Gain Coefficients\n");
		printf("7. Write Offset/Gain Coefficients To Flash\n");
		printf("8. Display Offset/Gain Coefficients In Memory\n");
	        printf("\nSelect: ");
	        scanf("%x",&item);

	        switch(item)
	        {
	          case 1: /* return to previous menu */
	            finished2++;
	          break;

	          case 2: /* Read Calibration Coefficients */
	            memset(&c_block236.IDbuf[0],0,sizeof(c_block236.IDbuf));	/* empty the buffer */
	            ReadFlashID236(&c_block236, &c_block236.IDbuf[0]);

	            if( (strstr( (const char *)&c_block236.IDbuf[0], (const char *)FlashIDString ) == NULL) )	/* AP236 ID */
	               printf("\n>>> Unable to read APBoard FLASH ID <<<\n");
	            else
	            {
	               printf("\n%s found, reading calibration coefficients...\n",&c_block236.IDbuf[0]);
	               rcc236(&c_block236); /* read the calibration coef. into an array */
	            }
	          break;

	          case 3: /* get gain */
			printf("\nEnter gain coefficient (HEX): ");
			scanf("%x", &temp);
			c_block236.ogc236[current_channel][range][GAIN] = (word)temp;
	          break;

	          case 4: /* get offset */
			printf("\nEnter offset coefficient (HEX): ");
			scanf("%x", &temp);
			c_block236.ogc236[current_channel][range][OFFSET] = (word)temp;
	          break;

	          case 5: /* Select new channel */
                        selectch236(&current_channel);
        	  break;

	          case 6: /* Cal channel */
			wro236(&c_block236, current_channel, (word)0x828F);	/* 0x828F is BTC for low endpoint */
			printf("\nEnter measured output value from DVM connected to channel %X: ",current_channel);
			scanf("%lf",&zero);
			wro236(&c_block236, current_channel, (word)0x7D70);	/* 0x7D70 is BTC for high endpoint */
			printf("\nEnter measured output value from DVM connected to channel %X: ",current_channel);
			scanf("%lf",&span);

			/* compute coefficients */
			if(span - zero == 0.0)	/* abort division by zero */
			  break;

			/* gain calculation */
			slope = (64880.0 - 655.0) / (span - zero);
			gcoef = 65536.0 * 16.0 * ((slope / (*c_block236.pIdealCode)[range][IDEALSLOPE]) - 1.0);
			gcoef += (gcoef < 0.0) ? -0.5 : 0.5; /* round */

			/* offset calculation */
			ocoef = ((655.0 - (slope * zero)) - (*c_block236.pIdealCode)[range][IDEALZEROSB]) * 16.0;
			ocoef += (ocoef < 0.0) ? -0.5 : 0.5; /* round */

			printf("\nOffset Coefficient = %04X",(word)ocoef);
			printf("\nGain   Coefficient = %04X",(word)gcoef);
			printf("\n\nDo you wish to update the gain/offset data arrays (Y or N) : ");
			scanf("%s",cmd_buff);
            		if( cmd_buff[0] == 'y' || cmd_buff[0] == 'Y' )
			{
			  c_block236.ogc236[current_channel][range][OFFSET] = (short)ocoef;
			  c_block236.ogc236[current_channel][range][GAIN] = (short)gcoef;
			}
        	  break;

	          case 7: /* Write Offset/Gain Coefficients To Flash */
			printf("\n                      >>> CAUTION! <<<\n");
			printf("This selection will overwrite ALL offset & gain coefficients\n");
			printf("stored in flash memory with the current offset & gain values\n");
			printf("you have established in this programs internal memory.\n");
			printf("\nAre you sure? (Y or N) : ");
			scanf("%s",cmd_buff);
            		if( cmd_buff[0] == 'y' || cmd_buff[0] == 'Y' )
            		{
		            if(WriteOGCoefs236(&c_block236))
				printf("\n>>> Error Writing Offset/Gain Coefficients To Flash <<<\n");
            		}
            		else
            		{
			    printf("\nFlash write aborted\n");
            		}
	          break;

	          case 8: /* Display offset & gain coefficients in memory */
	             for( i = 0; i < 8; i++ ) 
	             {
	               for( range = 0; range < 8; range++ )
	               {
		          printf("Ch %X Rng %X Offset %04X Gain %04X\n",i,range,
				(word)c_block236.ogc236[i][range][OFFSET],
				(word)c_block236.ogc236[i][range][GAIN]);
	               }
	             }
	          break;
       	    	}
	      }
	    }
            break;
      }   /* end of switch */
    }   /* end of while */

    if(c_block236.bAP)
        APClose(c_block236.nHandle);

    printf("\nEXIT PROGRAM\n");
    return(0);
}   /* end of main */



/*
{+D}
    SYSTEM:	    AP236 Software

    FILENAME:	    drvr236.c

    MODULE NAME:    scfg236 - set configuration block contents.

    VERSION:	    A

    CREATION DATE:  12/01/15

    CODED BY:	    FJM

    ABSTRACT:	    Routine which is used to enter parameters into
		    the Configuration Block.

    CALLING
	SEQUENCE:   scfg236(c_block236)
		    where:
			c_block236 (structure pointer)
			  The address of the configuration param. block

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
*/

void scfg236(struct cblk236 *c_blk, int channel)
{

/*
    DECLARE LOCAL DATA AREAS:
*/
    int item;		/* menu item variable */
    unsigned finished;	/* flags to exit loops */

/*
    ENTRY POINT OF ROUTINE:
*/

    finished = 0;
    while(!finished)
    {
	printf("\n\nConfiguration Parameters for Channel %X\n\n", channel);
	printf(" 1. Return to Previous Menu\n");
	printf(" 2. Board Pointer:	%lX\n",(unsigned long)c_blk->brd_ptr);
	printf(" 3. Parameter Mask:     %X\n",c_blk->opts.chan[channel].ParameterMask);
        printf(" 4. Output Update Mode: %X\n",c_blk->opts.chan[channel].UpdateMode);
        printf(" 5. Output Range:       %X\n",c_blk->opts.chan[channel].Range);
        printf(" 6. Power-up Voltage:   %X\n",c_blk->opts.chan[channel].PowerUpVoltage);
        printf(" 7. Thermal Shutdown:   %X\n",c_blk->opts.chan[channel].ThermalShutdown);
        printf(" 8. 5%% Overrange:       %X\n",c_blk->opts.chan[channel].OverRange);
        printf(" 9. Clear Voltage:      %X\n",c_blk->opts.chan[channel].ClearVoltage);
        printf("10. Data Reset:         %X\n",c_blk->opts.chan[channel].DataReset);
        printf("11. Full Device Reset:  %X\n",c_blk->opts.chan[channel].FullReset);
	printf("\nSelect: ");
	scanf("%d",&item);
	switch(item)
	{
	case 1: /* return to previous menu */
	    finished++;
	    break;

	case 2: /* board address */
	    printf("ADDRESS CAN NOT BE CHANGED\n");
	    break;

        case 3: /* Parameter Mask */
	    printf("Device Configuration Bit Mask %02X\nA set Bit Updates the Option\n",c_blk->opts.chan[channel].ParameterMask);
	    printf("Bit 0 Set = Output Range\n");
	    printf("Bit 1 Set = Power-up Voltage\n");
	    printf("Bit 2 Set = Thermal Shutdown\n");
	    printf("Bit 3 Set = 5%% Overrange\n");
	    printf("Bit 4 Set = Clear Voltage\n");
	    printf("Bit 5 Set = Output Update Mode\n");
	    printf("Bit 6 Set = Data Reset\n");
	    printf("Bit 7 Set = Full Device Reset\n");
	    c_blk->opts.chan[channel].ParameterMask = (int)(get_param() & 0xff);
	    break;

        case 4: /* Output Update Mode */
	    printf("0 - Transparent Mode\n");
	    printf("1 - Simultaneous Mode\n");
	    c_blk->opts.chan[channel].UpdateMode = (int)(get_param() & 1);
	    break;

        case 5: /* Output Range */
	    printf("0 -  -10V to +10V\n");
	    printf("1 -    0V to +10V\n");
	    printf("2 -   -5V to +5V\n");
	    printf("3 -    0V to +5V\n");
	    printf("4 - -2.5V to +7.5V\n");
	    printf("5 -   -3V to +3V\n");
	    printf("6 -    0V to +16V\n");
	    printf("7 -    0V to +20V\n");
	    c_blk->opts.chan[channel].Range = (int)(get_param() & 7);
	    break;

        case 6: /* Power-up Voltage */
	    printf("0 -  Zero Scale\n");
	    printf("1 -  Mid Scale\n");
	    printf("2 -  Full Scale\n");
	    c_blk->opts.chan[channel].PowerUpVoltage = (int)(get_param() & 3);
	    break;

        case 7: /* Thermal Shutdown */
	    printf("0 -  Disable\n");
	    printf("1 -  Enable\n");
	    c_blk->opts.chan[channel].ThermalShutdown = (int)(get_param() & 1);
	    break;

        case 8: /* 5% Overrange */
	    printf("0 -  Disable\n");
	    printf("1 -  Enable\n");
	    c_blk->opts.chan[channel].OverRange = (int)(get_param() & 1);
	    break;

        case 9: /* Clear Voltage */
	    printf("0 -  Zero Scale\n");
	    printf("1 -  Mid Scale\n");
	    printf("2 -  Full Scale\n");
	    c_blk->opts.chan[channel].ClearVoltage = (int)(get_param() & 3);

	    break;

        case 10: /* DataReset */
	    printf("0 -  Disable\n");
	    printf("1 -  Enable\n");
	    c_blk->opts.chan[channel].DataReset = (int)(get_param() & 1);
	    break;

        case 11: /* Full Device Reset */
	    printf("0 -  Disable\n");
	    printf("1 -  Enable\n");
	    c_blk->opts.chan[channel].FullReset = (int)(get_param() & 1);
	    break;
	}
    }
}


/*
{+D}
    SYSTEM:         Software

    FILENAME:       drvr236.c

    MODULE NAME:    selectch236 - Select channel.

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    FJM

    CODED BY:       FJM

    ABSTRACT:       Routine selects current channel.

    CALLING
        SEQUENCE:   selectch236(&current_channel)
		    where:
			current_channel (pointer)
			  The address of the current_channel variable to update

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
*/

void selectch236(int *current_channel)

{

/*
    DECLARE LOCAL DATA AREAS:
*/
    int cntr;             /* menu item variable */

/*
    ENTRY POINT OF ROUTINE:
*/
      printf("\n\nCurrent Channel: %X\n\n",*current_channel);
      printf("Enter New Channel Number (0 - 7): ");
      scanf("%x",&cntr);
      *current_channel = (cntr & 0x7);
}



/*
{+D}
    SYSTEM:         Software

    FILENAME:	    drvr236.c

    MODULE NAME:    psts236 - print board status information

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    FJM

    CODED BY:	    FJM

    ABSTRACT:	    Routine which is used to cause the "Read Board Status"
                    command to be executed and to print out the results to
                    the console.

    CALLING
	SEQUENCE:   psts236(&c_block)
		    where:
			c_block (structure pointer)
			  The address of the configuration param. block

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

*/

void psts236(c_blk)
struct cblk236 *c_blk;
{

/*
	DECLARE LOCAL DATA AREAS:
*/
	int item;	    /* menu item variable */
	unsigned finished;  /* flags to exit loops */
	int index;

/*
	ENTRY POINT OF ROUTINE:
*/

	finished = 0;
	while(!finished)
	{
	  rsts236(c_blk);	   /* Read Command */

	  printf("\n\nBoard Status Information");
	  printf("\nFirmware Revision:         %c",(char)c_blk->revision);
	  printf("\n\n1. Return to Previous Menu");
	  printf("\n2. Read Status Again\n3. FPGA Temp/Vcc Values\n");
	  printf("\nselect: ");
	  scanf("%d",&item);

	  switch(item)
	  {
	    case 1: /* return to previous menu */
	      finished++;
	    break;

	    case 3: /* display temp & VCC info from FPGA */
		  for( index = 0; index < 9; index++)
		  {
		    printf("Adr: %02X  FPGAData: %04X  ",
			  ((c_blk->FPGAAdrData[index] >> 16) & 0x7F),
			  ((c_blk->FPGAAdrData[index] >> 6) & 0x0FFF));

		    if((c_blk->FPGAAdrData[index] >> 16) & 3 ) /* Vcc */
			  printf("%7.3f V\n", ((c_blk->FPGAAdrData[index] >> 6) & 0x03FF) / 1024.0 * 3.0);
		    else            /* T deg C */
		      printf("%7.3f Deg C\n", ((c_blk->FPGAAdrData[index] >> 6) & 0x0FFF) * 503.975 / 1024.0 - 273.15);
		  }
	    break;
	  }
	}
}

