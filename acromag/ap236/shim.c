// shim contains a helper function that does type conversion
// within C to avoid making the Go type system angry
#include "../apcommon/apcommon.h"
#include "AP236.h"

APSTATUS GetAPAddress2(int nHandle, struct map236** pAddress)
{
	return (APSTATUS)GetAPAddress(nHandle, (long*)pAddress);
}


int Setup_board_cal(struct cblk236* c_block236)
{
	memset(&c_block236->IDbuf[0],0,sizeof(c_block236->IDbuf));	/* empty the buffer */
    ReadFlashID236(c_block236, &c_block236->IDbuf[0]);

    if( (strstr( (const char *)&c_block236->IDbuf[0], (const char *)FlashIDString ) == NULL) ) {/* AP220 or AP236 ID */
		  return -1;
	}
	rcc236(c_block236); /* read the calibration coef. into an array */
	return 0;
}
