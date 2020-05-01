// shim contains a helper function that does type conversion
// within C to avoid making the Go type system angry
#include "../apcommon/apcommon.h"
#include "AP236.h"

APSTATUS GetAPAddress2(int nHandle, struct map236** pAddress)
{
	return (APSTATUS)GetAPAddress(nHandle, (long*)pAddress);
}
