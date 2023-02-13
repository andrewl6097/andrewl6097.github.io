---
layout: post
title: 'Embedded Linux Part 1: The NXP i.MX6ULL and Buildroot'
date: '2023-02-12 21:15:00 -0800'
categories: []
comments: false
tags: ['Embedded Linux']
---
Taking a detour from my Outdoor Mic series, as I have a new toy that's arrived in the mail: an [i.MX 6ULL evaluation kit](https://www.nxp.com/design/development-boards/i-mx-evaluation-and-development-boards/evaluation-kit-for-the-i-mx-6ull-and-6ulz-applications-processor:MCIMX6ULL-EVK).  Here it is on my desk:

![p11](/assets/img/embedded-linux/IMG_1092.png)

I've [taken](https://twitter.com/andrew_lusk/status/1466247250602983427) [a](https://twitter.com/andrew_lusk/status/1467396002806046723) [run](https://twitter.com/andrew_lusk/status/1467267125660160002) [at](https://twitter.com/andrew_lusk/status/1467565814693785607) an embedded linux project before, when I made a little breakout board for the [Microchip SAM9X60](https://www.microchip.com/en-us/product/SAM9X60).  This was all inspired by this excellent article called ["So you want to build an embedded linux system?"](https://jaycarlson.net/embedded-linux/).  I really suggest giving it a read.  It's firmly in the genre I try to emulate, which is "this thing you think is too hard, is actually less hard than you might think if you just work through it".  I took the recommendation of starting with the SAM9X60 from there.

The challenge I had with it was that it wasn't quite fast enough, and it didn't have the peripherals that I wanted (which was really to make something I could play doom on).  I also super-ultra-hacked up the OS image, in non-repeatable ways that I have all lost as part of sweeping up unused crud from my AWS account.

I also have set a goal for myself that I want to make a PCB that includes DDR memory on it - which opens up options considerably.  I'm going to start working through that process, but in parallel, I want to work on the software, and use thie evaluation kit performs better (it ought to - a 900MHz Cortex-A7 should smoke a 600MHz ARM9).  The eval kit doesn't have a display - I've ordered one, it takes a non-standard 50-pin connector which is annoying - and since I want to run games on it, that'll hold me back eventually.  But I can get started.

Part of the value of writing this stuff down for me is so I don't lose the knowledge later.  The first step here will be building an OS image to sit on the SD card.  The evaluation kit comes with one, but I want a clean sheet of paper to start iterating on, working on boot time, configuring the right peripherals, etc.

The tool that I used before for this is called 'buildroot' - it's basically a 'make menuconfig' (if you've ever compiled a Linux kernel) but for a whole embedded OS, that spits out a .img file for an SD card at the end.  If you go to the [embedded linux article](https://jaycarlson.net/embedded-linux/) I linked above, and CTL-F for 'yocto & buildroot', you'll find a great explainer on two major approaches to building OS images for embedded linux systems.  The card that came with the EVK had an image built with Yocto, but I'm a little more familiar with buildroot, and I'm going to stick with it.

Buildroot has support for this exact eval kit, and there's a [readme](https://fossies.org/linux/buildroot/board/freescale/imx6ullevk/readme.txt) that tells me just what to do to get a base image.

Downloading the latest tarball from the [buildroot site](https://buildroot.org/downloads/), I just run

    make imx6ullevk_defconfig

and then

    make -j12

This basically downloads a zillion linux packages, including the kernel, and cross-compiles them all for the target platform.  It installs a [basic bootloader](https://u-boot.readthedocs.io/en/latest/) which itself knows how to set up Linux for success with device tree files, etc.  On my box - computers are fast now! - this takes about 15 minutes to drop 'sdcard.img' into output/images:

```
INFO: vfat(boot.vfat): cmd: "mkdosfs  -n 'boot' '/home/andrew/buildroot-2022.11/output/images/boot.vfat'" (stderr):
mkfs.fat: Warning: lowercase labels might not work properly on some systems
INFO: vfat(boot.vfat): adding file 'imx6ull-14x14-evk.dtb' as 'imx6ull-14x14-evk.dtb' ...
INFO: vfat(boot.vfat): cmd: "MTOOLS_SKIP_CHECK=1 mcopy -sp -i '/home/andrew/buildroot-2022.11/output/images/boot.vfat' '/home/andrew/buildroot-2022.11/output/images/imx6ull-14x14-evk.dtb' '::'" (stderr):
INFO: vfat(boot.vfat): adding file 'zImage' as 'zImage' ...
INFO: vfat(boot.vfat): cmd: "MTOOLS_SKIP_CHECK=1 mcopy -sp -i '/home/andrew/buildroot-2022.11/output/images/boot.vfat' '/home/andrew/buildroot-2022.11/output/images/zImage' '::'" (stderr):
INFO: hdimage(sdcard.img): adding partition 'u-boot' from 'u-boot-dtb.imx' ...
INFO: hdimage(sdcard.img): adding partition 'boot' (in MBR) from 'boot.vfat' ...
INFO: hdimage(sdcard.img): adding partition 'rootfs' (in MBR) from 'rootfs.ext2' ...
INFO: hdimage(sdcard.img): adding partition '[MBR]' ...
INFO: hdimage(sdcard.img): writing MBR
```

Let's see if this boots:

```
U-Boot 2021.10 (Feb 09 2023 - 19:44:38 -0800)

CPU:   Freescale i.MX6ULL rev1.1 900 MHz (running at 396 MHz)
CPU:   Commercial temperature grade (0C to 95C) at 35C
Reset cause: POR
Model: Freescale i.MX6 UltraLiteLite 14x14 EVK Board
Board: MX6ULL 14x14 EVK
DRAM:  512 MiB
MMC:   FSL_SDHC: 0, FSL_SDHC: 1
Loading Environment from MMC... *** Warning - bad CRC, using default environment

In:    serial
Out:   serial
Err:   serial
Net:   eth1: ethernet@20b4000 [PRIME]Get shared mii bus on ethernet@2188000
, eth0: ethernet@2188000
Hit any key to stop autoboot:  0 
switch to partitions #0, OK
mmc1 is current device
switch to partitions #0, OK
mmc1 is current device
Failed to load 'boot.scr'
9660912 bytes read in 412 ms (22.4 MiB/s)
Booting from mmc ...
31724 bytes read in 3 ms (10.1 MiB/s)
Kernel image @ 0x82000000 [ 0x000000 - 0x9369f0 ]
## Flattened Device Tree blob at 83000000
   Booting using the fdt blob at 0x83000000
   Using Device Tree in place at 83000000, end 8300abeb

Starting kernel ...

.
.
.

[    8.713786] Freeing unused kernel image (initmem) memory: 1024K
[    8.721094] Run /sbin/init as init process
[    9.050364] EXT4-fs (mmcblk1p2): re-mounted. Opts: (null). Quota mode: none.
Starting syslogd: OK
Starting klogd: OK
Running sysctl: OK
Initializing random number generator: OK
Saving random seed: OK
Starting network: OK

Welcome to Buildroot
buildroot login:
```

Nice!  Though I dont love the 'running at 396MHz'...I'm going to have to look into that.

This was definitely a lot easier than the SAM9X60 board I put together - though that's in large part, I'm sure, due to starting from a known platform (the i.MX 6ULL EVK) which buildroot has support for.  When and if I have my own i.MX6ULL PCB - there'll be more to do.  But, I'm going to start by pretending I do have my own board, and working to integrate it with buildroot in 'the right way', that's sustainable, and that I can check into source control.

Let's work to get a couple of basics working.  First off - networking.  eth0 is detected, but doesn't DHCP.  Walking through 'make menuconfig', there's an option for "Network interface to configure through DHCP", which I set to "eth0".

Turns out this doesnt work - already something to debug.  Grepping through the buildroot package for the config variable I set to eth0 - BR2_SYSTEM_DHCP - I get the sense that maybe it's meant to integrate with a proper init system.  Maybe even systemd, not that I truly understand systemd.  Anyway - I'm probably going to want something more sophisticated than the default BusyBox init system, so I switch it over to systemd.

Another 'make' and copying the sdcard.img over, and fiddling with the freaking annoying sd card tray on the EVK...still no.  It's still using init from busybox.

Now trying a 'make clean' - and that'll do it.  I remember this now - buildroot is pretty easy to use, but basically, if you ask it to do anything big, it's not *that* smart (and I'm sure it's limited by what 'make' can do under the covers).

Now we have a real systemd and eth0 comes up just fine on boot:

```
[  OK  ] Reached target System Initialization.
[  OK  ] Started Daily Cleanup of Temporary Directories.
[   24.106273] Micrel KSZ8081 or KSZ8091 20b4000.ethernet-1:01: attached PHY driver (mii_bus:phy_addr=20b4000.ethernet-1:01, irq=POLL)
[  OK  ] Reached target Timer Units.
[  OK  ] Listening on D-Bus System Message Bus Socket.
[  OK  ] Reached target Socket Units.
[  OK  ] Reached target Basic System.
[  OK  ] Started D-Bus System Message Bus.
[  OK  ] Started Serial Getty on ttymxc0.
[  OK  ] Reached target Login Prompts.
[  OK  ] Reached target Multi-User System.
[   25.155184] ov5640 1-003c: supply DOVDD not found, using dummy regulator
[   25.273665] ov5640 1-003c: supply AVDD not found, using dummy regulator
[   25.343467] ov5640 1-003c: supply DVDD not found, using dummy regulator
[   25.551668] ov5640 1-003c: ov5640_read_reg: error: reg=300a
[   25.557318] ov5640 1-003c: ov5640_check_chip_id: failed to read chip identifier

Welcome to Andrew's Embedded Linux!
buildroot login: [   27.313127] fec 20b4000.ethernet eth0: Link is Up - 100Mbps/Full - flow control rx/tx
[   27.401334] IPv6: ADDRCONF(NETDEV_CHANGE): eth0: link becomes ready
[   28.147416] systemd-journald[148]: Oldest entry in /var/log/journal/c8321436d2ae4e1db5ae98bb47a46390/system.journal is older than the configured file retention duration (1month), suggesting rotation.
[   28.291112] systemd-journald[148]: /var/log/journal/c8321436d2ae4e1db5ae98bb47a46390/system.journal: Journal header limits reached or header out-of-date, rotating.
[   30.123311] rtc rtc0: Timeout trying to get valid LPSRT Counter read
[   38.915655] VSD_3V3: disabling
[   38.919493] can-3v3: disabling
[   69.657902] cfg80211: failed to load regulatory.db
[   69.677015] imx-sdma 20ec000.sdma: external firmware not found, using ROM firmware
root
# ifconfig 
eth0      Link encap:Ethernet  HWaddr 00:04:9F:07:4C:9A  
          inet addr:192.168.115.228  Bcast:192.168.115.255  Mask:255.255.255.0
          inet6 addr: fe80::204:9fff:fe07:4c9a/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:36 errors:0 dropped:0 overruns:0 frame:0
          TX packets:27 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000 
          RX bytes:3890 (3.7 KiB)  TX bytes:2600 (2.5 KiB)

lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000 
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)

# 
```

Going to call it a night with that.  The next step?  *Some* kind of video output, maybe with X11, unless the LCD screen from eBay arrives first.
