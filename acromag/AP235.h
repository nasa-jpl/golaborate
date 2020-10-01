
/*
{+D}
    SYSTEM:         Software for AP235

    FILE NAME:      AP235.h

    VERSION:        A

    CREATION DATE:  12/01/15

    DESIGNED BY:    FM

    CODED BY:       FM

    ABSTRACT:       This module contains the definitions and structures
                    used by the AP235 library.

    CALLING
        SEQUENCE:

    MODULE TYPE:    header file

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
    DEFINITIONS:
*/

/* ////////////////////////////////////////////////////////////////// */
/* Select CPU type that corresponds to your hardware.                 */
/* Default is A64 - #define BUILDING_FOR_A64 0 to build for A64 CPU.  */
/* Comment out '#define BUILDING_FOR_A64 0' to build for A32 CPU.     */
/* ////////////////////////////////////////////////////////////////// */
#define BUILDING_FOR_A64 0




/* A235 board */
#define AP235 (word)0x701D        /* AP235 device ID */
#define DEVICE_NAME "ap235_"      /* name of device */
#define FlashIDString "AP235"     /* Flash ID String value */


#define FlashCoefficientMemoryAddress	0x3FE000 /* starting address ( channel 0) */
#define FlashCoefficientIDString	0x3FEFF0 /* ID string starting address */


/* DAC commands */
#define SMWrite		1
#define DACUpdate	2
#define TMWrite		3
#define WriteControl	4
#define DataResetWrite	7
#define FullResetWrite	0xF

/* DAC update modes */
#define DAC_DA		0	/* DAC Direct Access */
#define DAC_CONT	1	/* DAC Continuous mode */
#define DAC_FIFO	2	/* DAC FIFO mode */
#define DAC_SB		3	/* DAC Single Burst mode */
#define DAC_FIFO_DMA	4	/* DAC FIFO DMA mode */

/* Channel status bits */
#define FIFO_empty	(1 << 0)
#define FIFO_half_full	(1 << 1)
#define FIFO_full	(1 << 2)
#define FIFO_underflow	(1 << 3)
#define BS_clear	(1 << 4)


#define IDEALZEROSB	0	/* indices to elements */
#define IDEALZEROBTC	1
#define IDEALSLOPE	2
#define ENDPOINTLO	3
#define ENDPOINTHI	4
#define CLIPLO		5
#define CLIPHI		6
#define OFFSET		0
#define GAIN		1


#define AXI_RAM_BASE	0xA000	/* base address of scatter-gather descriptor list RAM memory space */
#define AXIBAR_0	0x80000	/* AXIBAR_0 base address */

#define DMAMAX_TRIES	300000
#define MAXSAMPLES	4096	/* individual channel data buffer size */
#define MAX_MEMORY_PAGES (16 * 2) + 2 /* 2 pages per channel x 16 channels */




/* DMA register bits */
#define ScatterGather			(1 << 3)		/* DMA mode */
#define DMAInterruptPending		(1 << 16)
#define DMAInterruptEnable		(1 << 16)
#define MasterInterruptEnable		(3)
#define MasterInterruptDisable		(0)

#define DMATransferComplete		(1 << 1)
#define DMAReset			(1 << 2)
#define DMAKeyHoleWrite			(1 << 5)
#define DMAInterruptOnCompleteEnabled	(1 << 12)
#define DMAInterruptOnDelayTimerEnabled	(1 << 13)

/* Interrupt types */
#define FIFO_SBURST	1



struct scatterAP235list		/* structure for scatter-gather DMA */
{
    uint32_t NxtDescPtrLo;	/* Next Descriptor Pointer */
    uint32_t NxtDescPtrHi;
    uint32_t SrcAddressLo;	/* Source Address */
    uint32_t SrcAddressHi;
    uint32_t DstAddressLo;	/* Destination Address */
    uint32_t DstAddressHi;
    uint32_t Control;		/* Bytes to Transfer */
    uint32_t Status;		/* Status Register  */
    uint32_t AddrTranslationHi;
    uint32_t AddrTranslationLo;	/* Address Translation */
    struct page	*page;		/* mapped user space page pointers */
#ifndef BUILDING_FOR_A64
    uint32_t alignment;		/* keep structure aligned */
#endif
    uint32_t Unusedsgl[4];	/* Next Descriptor Pointers 64 byte aligned */
};



/*
    Defined below is the memory map template for AP235 Boards.
    This structure provides access to the various registers on the board.
*/

struct mapap235			/* Memory map of the I/O board space */
{
    /* AXI Central DMA Register Space 00000000 - 00000FFF */
    uint32_t CDMAControlRegister;
    uint32_t CDMAStatusRegister;
    uint32_t CDMADescriptorPointerRegister;
    uint32_t CDMADescriptorPointerRegisterHi;
    uint32_t CDMATailDescriptorPointerRegister;
    uint32_t CDMATailDescriptorPointerRegisterHi;
    uint32_t CDMASourceAddressRegister;
    uint32_t CDMASourceAddressRegisterHi;
    uint32_t CDMADestinationAddressRegister;
    uint32_t CDMADestinationAddressRegisterHi;
    uint32_t CDMABytesToTransferRegister;
    unsigned char AXI_CDMAUnused[0xFD4];

    /* PCIe AXI Bridge Control Register Space 00001000 - 00001FFF */
    unsigned char PCIeAXIBridgeControlRsv0[0x144];
    uint32_t AXIBridgePHYStatusControl;
    unsigned char PCIeAXIBridgeControlRsv1[0xC0];
    uint32_t AXIBAR2PCIEBAR_0U;
    uint32_t AXIBAR2PCIEBAR_0L;
    unsigned char PCIeAXIBridgeControlRsv2[0xDF0];
	
    /* AXI Interrupt Controller Register Space 00002000 - 00002FFF */
    uint32_t AXI_InterruptStatusRegister;
    uint32_t AXI_InterruptPendingRegister;
    uint32_t AXI_InterruptEnableRegister;
    uint32_t AXI_InterruptAcknowledgeRegister;
    uint32_t AXI_SetInterruptEnableRegister;
    uint32_t AXI_ClearInterruptEnableRegister;
    uint32_t AXI_InterruptVectorRegister;
    uint32_t AXI_MasterEnableRegister;
    unsigned char AXI_InterruptControllerRsv1[0xFE0];

    /* XADC System Monitor Register Space 00003000 - 00003FFF */
    unsigned char AXISysmonCore[0x200];	/* Core Registers */
    uint32_t XW_Temperature;		/* temperature Data Register (200) */
    uint32_t XW_VCCInt;			/* supply monitor Data Register (204) */
    uint32_t XW_VCCAux;			/* supply monitor Data Register (208) */
    unsigned char AXISysmonCore1[0x74];	/* Reserved */
    uint32_t XW_MAXTemperature;		/* MAXtemperature Data Register (280) */	
    uint32_t XW_MAXVCCInt;		/* MAXsupply monitor Data Register (284) */
    uint32_t XW_MAXVCCAux;		/* MAXsupply monitor Data Register (288) */
    uint32_t XW_invalid;		/* not used */
    uint32_t XW_MINTemperature;		/* MINtemperature Data Register (290) */	
    uint32_t XW_MINVCCInt;		/* MINsupply monitor Data Register (294) */
    uint32_t XW_MINVCCAux;		/* MINsupply monitor Data Register (298) */	
    unsigned char AXISysmonCore2[0xD64];/* Reserved */

    /* Firmware Revision Register Space 00004000 - 00004FFF */
    uint32_t FirmwareRevision;		/* 31:0 */
    unsigned char FirmwareRevision_Reserved[0xFFC];
	
    /* AXI_QSPI Register Space 00005000 - 00005FFF */
    unsigned char QSPI_Reserved[0x20];
    uint32_t QSPI_IPISR;		/* Interrupt status register */
    unsigned char QSPI_Reserved1[0x1C];
    uint32_t QSPI_SR;			/* Software reset register */
    unsigned char QSPI_Reserved2[0x1C];
    uint32_t QSPI_SPICR;		/* SPI control register */
    uint32_t QSPI_SPISR;		/* SPI status register */
    uint32_t QSPI_SPIDTR;		/* SPI data transmit register */
    uint32_t QSPI_SPIDRR;		/* SPI data receive register */
    uint32_t QSPI_SPISSR;		/* SPI Slave select register */
    uint32_t QSPI_SPITFOR;		/* Transmit FIFO occupancy register */
    uint32_t QSPI_SPIRFOR;		/* Receive FIFO occupancy register */
    unsigned char QSPI_Reserved3[0xF84];/* Reserved */

    /* Location Register Space 00006000 - 00006FFF */
    uint32_t LocationRegister;		/* Bits 7:0 */
    unsigned char LocationRegisterReserved[0xFFC];

    /* Reserved Register Space1 00007000 - 00009FFF */
    unsigned char Reserved1[0x3000];

    /* Scatter-gather list block RAM 0000A000 - 0000BFFF used for ping-pong buffering of DAC channel data */
    struct sgchdesc235
    {
      /* structure for scatter-gather DMA first page translation register lo descriptor */
      struct scatterAP235list fptrlo;
      /* structure for scatter-gather DMA first page translation register hi descriptor */
      struct scatterAP235list fptrhi;
      /* structure for scatter-gather DMA first page DAC data descriptor */
      struct scatterAP235list fpdata;

      /* structure for scatter-gather DMA second page translation register lo descriptor */
      struct scatterAP235list sptrlo;
      /* structure for scatter-gather DMA second page translation register hi descriptor */
      struct scatterAP235list sptrhi;
      /* structure for scatter-gather DMA second page DAC data descriptor */
      struct scatterAP235list spdata;
    } CHAN[16];
    unsigned char ReservedScatterGatherRAM[0x7FF];

    /* Reserved Register Space2 0000C000 - 0003FFFF */
    unsigned char Reserved2[0x34000];

    /* DAC Register Space 00040000 - 0005FFFF */
    struct
    {
      uint32_t StartAddr;		/* 31:0 Channel Start Address register */
      uint32_t EndAddr;			/* 31:0 Channel End Address register */
      uint32_t Fifo;			/* 31:0 Channel FIFO register */
      uint32_t DAC_Reserved1;
      uint32_t Control;			/* Channel Control register */
      uint32_t Status;			/* Channel Status register */
      uint32_t DirectAccess;		/* Channel Direct Access register */
      uint32_t DAC_Reserved2;
    } DAC[16];				/* 16 DAC channels */
    uint32_t CommonControl;		/* Common Control register */
    uint32_t TimerDivider;		/* Timer Divider register */
    uint32_t SoftwareTrigger;		/* Software Trigger register */
    unsigned char DAC_Reserved3[0x1FDF4];/* Reserved */

    /* Sample Memory Space 00060000 - 0007FFFF */
    word SampleMemory[0xFFFF];
};



struct chops235 /* Channel Control Register Options */
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
    int OpMode;
    int TriggerSource;
    int UnderflowClear;
    int InterruptSource;
  }chan[16];
};



/*
    Defined below is the structure which is used to hold the board's configuration information.
*/

struct cblk235
{
    struct mapap235 *brd_ptr;	/* pointer to base address of board */
    APDATA_STRUCT* pAP;		/* pointer to AP data structure */
    uint32_t FPGAAdrData[10];	/* FPGA address & data order:0,1,2,20 thru 26 */
    int nHandle;		/* handle to an open board */
    BOOL bAP;			/* flag indicating a open board */
    BOOL bInitialized;          /* flag indicating ready to talk to board */
    struct chops235 opts;	/* DAC control register options */
    short ogc235[16][8][2];	/* offset & gain correction pairs[2] for each range[8] for each channel[16] */
    double (*pIdealCode)[8][7];	/* pointer to Ideal Zero, Slope, endpoint, and clip constants */
    unsigned char IDbuf[32];	/* storage for APxxx ID string */
    uint32_t ChStatus[16];	/* Channel status values for each output */
    uint32_t TimerDivider;	/* Timer Register Value */
    uint32_t TriggerDirection;	/* Trigger direction */
    uint32_t revision;		/* Firmware Revision */
    ushort location;		/* AP location */
    uint32_t SampleCount[16];	/* number of samples in buffer for each Channel */
    short ideal_buf[16][MAXSAMPLES]; /* allocate ideal data storage area */
    short (*pcor_buf)[16][MAXSAMPLES]; /* pointer to allocated corrected data storage area */
    short *head_ptr[16];	/* head pointer of write buffer */
    short *tail_ptr[16];	/* tail pointer of write buffer */
    short *current_ptr[16];	/* current data pointer of write buffer */
};


/* Declare functions called */
void rsts235(struct cblk235 *c_blk);		/* read board status */
void psts235(struct cblk235 *c_blk);		/* print status */
int ReadFlashID235(struct cblk235 *c_blk, unsigned char *p );	/*read flash ID */
int rcc235( struct cblk235 *c_blk );		/* Read gain & offset data from board FLASH */
int WriteOGCoefs235(struct cblk235 *c_blk);
void selectch235(int *current_channel);
void scfg235(struct cblk235 *c_blk, int channel);
void DMA_sandbox(struct cblk235 *c_blk, int channel);
void cnfg235(struct cblk235 *c_blk, int channel); /* configure channel */
void fifowro235( struct cblk235 *c_blk, int channel ); /* performs the write output function */
void fifodmawro235(struct cblk235 *c_blk, int channel);/* performs the DMA output function */
void cd235(struct cblk235 *c_blk, int channel, double *fb);	/* correct DAC output data */
void simtrig235(struct cblk235 *c_blk);





