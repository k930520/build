cp build_hi3798/emmc_bootargs.txt SDK/configs/hi3798mv100/prebuilts/emmc_bootargs.txt
cp build_hi3798/emmc_partitions.xml SDK/configs/hi3798mv100/prebuilts/emmc_partitions.xml
mkdir compress
cd SDK
source ./env.sh
cp configs/hi3798mv100/hi3798mdmo1g_hi3798mv100_cfg.mak ./cfg.mak
make prebuilts
tar -czvf ../compress/hi3798.tar.gz ./out/hi3798mv100/hi3798mdmo1g/image/emmc_image/*
