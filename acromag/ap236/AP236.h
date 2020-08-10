
/*
{+D}
    SYSTEM:	    Software for AP236

    FILE NAME:	    AP236.h

    VERSION:	    A

    CREATION DATE:  12/01/15

    DESIGNED BY:    F.M.

    CODED BY:	    F.M.

    ABSTRACT:      This module contains the definitions and structures
                   used by the AP236 library.

    CALLING
    SEQUENCE:

    MODULE TYPE:    header file

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

    This module contains the definitions and structures used by the library.
*/

/*
    DEFINITIONS:
*/


/*
	AP236 board type
*/

#define AP236 (word)0x702B         /* AP236 device ID */
#define DEVICE_NAME "ap236_"       /* name of device */
#define FlashIDString "AP236"	   /* Flash ID String value */



#define FlashCoefficientMemoryAddress	0x3FE000 /* starting address ( channel 0) */
#define FlashCoefficientIDString	0x3FEFF0 /* ID string starting address */


/* DAC commands */
#define SMWrite		1
#define DACUpdate	2
#define TMWrite		3
#define WriteControl	4
#define DataResetWrite	7
#define FullResetWrite	0xF



#define IDEALZEROSB	0	/* indices to elements */
#define IDEALZEROBTC	1
#define IDEALSLOPE	2
#define ENDPOINTLO	3
#define ENDPOINTHI	4
#define CLIPLO		5
#define CLIPHI		6
#define OFFSET		0
#define GAIN		1


/*
    STRUCTURES:

    Defined below is the memory map template for the AP236 Board.
    This data structure provides access to the various registers on the board.
*/


struct map236		/* Memory map of the I/O board */
{
    uint32_t Reserved1[2];
    uint32_t dac_reg[8];		/* DAC registers 0 - 7*/
    uint32_t Rsvdac_reg[8];
    uint32_t SimultaneousMode;
    uint32_t SimultaneousOutputTrigger;	/* trigger register */
    uint32_t Reserved2;
    uint32_t DACWriteStatus;		/* DAC Write Status */
    uint32_t DACResetControl;		/* DAC Reset Control */
    unsigned char Reserved3[0x2C];
    uint32_t XDAC_StatusControl;
    uint32_t XDAC_AddressReg;
    unsigned char Reserved4[0x0170];
    uint32_t FirmwareRevision;		/* 31:0 */
    uint32_t FLASHData;			/* 7:0 */
    uint32_t FlashChipSelect;		/* bit 0 */
};

struct chops236 /* Channel Control Register Options */
{
  struct
  {
    int Range;
    int PowerUpVoltage;
    int ThermalShutdown;
    int OverRange;
    int ClearVoltage;
    int UpdateMode;
    int DataReset;
    int FullReset;
    int ParameterMask;
  }chan[8];
};

/*
    Defined below is the structure which is used to hold the board's configuration information.
*/

struct cblk236
{
    struct map236 *brd_ptr;	/* pointer to base address of board */
    uint32_t FPGAAdrData[10];	/* FPGA address & data order:0,1,2,20 thru 26 */
    int nHandle;	        /* handle to an open board */
    BOOL bAP;			/* flag indicating a board is open */
    BOOL bInitialized;		/* flag indicating board is Initialized */
    struct chops236 opts;	/* DAC control register options */
    short ogc236[8][8][2];	/* storage for offset & gain correction pairs[2] for each range[8] for each channel[8] */
    double (*pIdealCode)[8][7];	/* pointer to Ideal Zero, Slope, endpoint, and clip constants */
    short cor_buf[8];		/* corrected buffer start */
    short ideal_buf[8];		/* ideal buffer start */
    unsigned char IDbuf[32];	/* storage for AP236 ID string */
    uint32_t revision;		/* Firmware Revision */
};

/*
    DECLARE MODULES CALLED:
*/

int rcc236( struct cblk236 *c_blk );				/* read gain/offset information */
void wro236(struct cblk236 *c_blk, int channel, word data);	/* performs the write output function */
void cd236(struct cblk236 *c_blk, int channel, double Volts);	/* correct DAC output data */
void scfg236(struct cblk236 *c_blk, int channel);
void selectch236(int *current_channel);
void cnfg236(struct cblk236 *c_blk, int channel); /* configure channel */
void simtrig236(struct cblk236 *c_blk);
void psts236(struct cblk236 *c_blk);
void rsts236(struct cblk236 *c_blk);
int ReadFlashID236(struct cblk236 *c_blk, unsigned char *p );
int WriteOGCoefs236(struct cblk236 *c_blk);

