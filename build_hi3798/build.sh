cp build/build_hi3798/emmc_bootargs.txt SDK/configs/hi3798mv100/prebuilts/emmc_bootargs.txt
cp build/build_hi3798/emmc_partitions.xml SDK/configs/hi3798mv100/prebuilts/emmc_partitions.xml
mkdir compress
cd SDK
sh server_install.sh
cp configs/hi3798mv100/hi3798mdmo1g_hi3798mv100_cfg.mak ./cfg.mak
source ./env.sh
make menuconfig
make build -j4 2>&1  | tee -a buildlog.txt
tar -czvf ../compress/hi3798.tar.gz ./out/hi3798mv100/hi3798mdmo1g/image/emmc_image/*
