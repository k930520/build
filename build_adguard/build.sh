appName="adguard"

PreBuildRelease() {
  mkdir -p "build"
  cd adguard
  go mod tidy
}

BuildReleaseLinuxMuslArm() {
  BASE="https://musl.cc/"
  FILES=(armv7l-linux-musleabihf-cross)
  for i in "${FILES[@]}"; do
    url="${BASE}${i}.tgz"
    curl -L -o "${i}.tgz" "${url}"
    sudo tar xf "${i}.tgz" --strip-components 1 -C /usr/local
    rm -f "${i}.tgz"
  done
  OS_ARCHES=(linux-musleabihf-armv7l)
  CGO_ARGS=(armv7l-linux-musleabihf-gcc)
  GOARMS=('7')
  for i in "${!OS_ARCHES[@]}"; do
    os_arch=${OS_ARCHES[$i]}
    cgo_cc=${CGO_ARGS[$i]}
    arm=${GOARMS[$i]}
    echo building for ${os_arch}
    export GOOS=linux
    export GOARCH=arm
    export CC=${cgo_cc}
    export GOARM=${arm}
	export CGO_ENABLED=1
    go build -o ../build/$appName-$os_arch .
  done
}

MakeRelease() {
  cd ../build
  mkdir compress
  for i in $(find . -type f -name "$appName-linux-*"); do
    cp "$i" adguard
	upx -9 adguard
    tar -czvf compress/"$i".tar.gz adguard
  done
  cd ../
}

if [ "$1" = "release" ]; then
  PreBuildRelease
  BuildReleaseLinuxMuslArm
  MakeRelease
else
  echo -e "Parameter error"
fi
