#!/bin/bash
# 
dd if=/dev/zero of=ubuntu-rootfs.img bs=1M count=2048
sudo  mkfs.ext4  ubuntu-rootfs.img
rm -r rootfs
mkdir  rootfs
sudo mount ubuntu-rootfs.img rootfs/
sudo cp -rfp ubuntu-rootfs/*  rootfs/
sudo umount rootfs/
e2fsck -p -f ubuntu-rootfs.img
resize2fs  -M ubuntu-rootfs.img
