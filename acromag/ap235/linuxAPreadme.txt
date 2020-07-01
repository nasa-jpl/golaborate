
Acromag, Inc.
30765 S. Wixom Rd., P.O. Box 437
Wixom, Michigan  48393-7037  U.S.A.
E-mail: support@acromag.com
Telephone (248) 295-0310
FAX (248) 624-9234

    .-.
    /v\ 
   // \\
  /(   )\
   ^^-^^

Fedora 29
Kernel 4.20.14-200

Acromag: APSW-API-LNX, 9500491, Rev. B

A note about out-of-tree loadable kernel modules

Acromag developed a set of strategies, code, and testing methods to help us
reuse code, maintain quality, and maximize our testing resources in order
to get the best quality product to our customers in the shortest amount of time.
To accomplish this, out-of-tree loadable kernel modules have been
developed over the past ten years. Recently, out-of-tree loadable kernel
modules have become an unpopular topic with core Linux kernel developers.
Our Open Source out-of-tree drivers generally work with all kernel releases
back to 2.4 and many users still want a driver that will support the newest
hardware on older kernels making out-of-tree drivers a necessary business
solution for hardware vendors.
With kernel releases 3.4 and newer, loading any out-of-tree driver module
will taint the kernel with a debug message similar to this:
"Disabling lock debugging due to kernel taint".  It is not a kernel bug,
but expected behavior. The taint flags (multiple flags may be pending)
may be examined using the following:

cat/proc/sys/kernel/tainted
4096 = An out-of-tree module has been loaded.

If an out-of-tree driver is merged into the kernel (i.e. add the
sources and update the kernel Makefiles) and build it as part of the
kernel build, the taint and taint message will go away.


This media contains library routines for Acromag I/O Boards.
Following is a table that indicates which Acromag I/O Boards are
supported and in what subdirectory the support files may be found:

Subdirectory       | Boards Supported
-------------------+---------------------------------------------
AP220              |  AP220-16.
AP225              |  AP225-16.
AP226              |  AP226-8.
AP231              |  AP231-16.
AP235              |  AP235-16.
AP236              |  AP236-8.
AP323              |  AP323.
AP341              |  AP341-16.
AP342              |  AP341-12.
AP408              |  AP408.
AP418              |  AP418.
AP445              |  AP445.
AP440              |  AP440-1, AP440-2, AP440-3.
AP441              |  AP441-1, AP441-2, AP441-3.
AP471              |  AP471.
AP48X              |  AP482, AP483, AP484.
AP50x_AP51x_AP52x  |  AP500, AP512, AP513, AP520, AP521, AP522.
AP560              |  AP560.
APA7X0X            |  APA7201/2/3/4, APA7501/2/3/4.



Also included in each subdirectory is an "information" file
which contains a list and detailed explanation of all of
the program files which make up each library.  For example,
the information file for the AP408 board is named "info408.txt".

