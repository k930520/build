appName="routedns"

PreBuildRelease() {
  mkdir -p "build"
  cd routedns
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
    go build -o ../build/$appName-$os_arch .
    wget https://www.php.net/distributions/php-8.4.3.tar.gz
    sudo tar xf php-8.4.3.tar.gz
    cd php-8.4.3
    ./configure \
        --enable-embed \
        --enable-zts \
        --disable-zend-signals \
        --enable-zend-max-execution-timers
    make -j"$(getconf _NPROCESSORS_ONLN)"
    sudo make install
    cd ../
    wget https://github.com/caddyserver/xcaddy/releases/download/v0.4.4/xcaddy_0.4.4_linux_amd64.tar.gz
    sudo tar xf xcaddy_0.4.4_linux_amd64.tar.gz
    CGO_ENABLED=1 \
    CGO_CFLAGS=$(php-config --includes) \
    CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" \
    ./xcaddy build \
        --with github.com/dunglas/frankenphp/caddy \
	--with github.com/dunglas/mercure/caddy \
 	--with github.com/dunglas/vulcain/caddy
  done
}

MakeRelease() {
  cd ../build
  mkdir compress
  for i in $(find . -type f -name "$appName-linux-*"); do
    cp "$i" routedns
	upx -9 routedns
    tar -czvf compress/"$i".tar.gz routedns
  done
  upx -9 ../routedns/caddy
  tar -czvf compress/caddy.tar.gz ../routedns/caddy
  cd ../
}

if [ "$1" = "release" ]; then
  PreBuildRelease
  BuildReleaseLinuxMuslArm
  MakeRelease
else
  echo -e "Parameter error"
fi
