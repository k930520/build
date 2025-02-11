cp -r build_alist/alist* .

appName="alist"
builtAt="$(date +'%F %T %z')"
goVersion=$(go version | sed 's/go version //')
cd alist
git tag -d beta
version=$(git describe --abbrev=0 --tags)
cd ../
cd alist-web
webVersion=$(git describe --abbrev=0 --tags)
cd ../

echo "backend version: $version"
echo "frontend version: $webVersion"

ldflags="\
-w -s \
-X 'github.com/alist-org/alist/v3/internal/conf.BuiltAt=$builtAt' \
-X 'github.com/alist-org/alist/v3/internal/conf.GoVersion=$goVersion' \
-X 'github.com/alist-org/alist/v3/internal/conf.Version=$version' \
-X 'github.com/alist-org/alist/v3/internal/conf.WebVersion=$webVersion' \
"

FetchWebRelease() {
  cd alist-web
  sed -i 's/Aliyundrive(Open)/Aliyundrive(Open|Share)/' src/pages/home/previews/index.ts
  unzip zh-CN.zip
  sed -i '216d' src/lang/zh-CN/drivers.json
  sed -i '/^  "AliyundriveShare".*: {$/r drivers.txt' src/lang/zh-CN/drivers.json
#  cp -r drivers.json src/lang/zh-CN/drivers.json
  pnpm install
  node ./scripts/i18n.mjs
  rm -rf src/lang/en
  pnpm build
  rm -rf ../alist/public/dist
  cp -rf dist ../alist/public
  cd ../
}

PreBuildRelease() {
  mkdir -p "build"
  cd alist
#  sed -i 's/SetRetryCount(3)/SetRetryCount(1)/' drivers/base/client.go
#  rm -f go.*
#  go mod init github.com/alist-org/alist/v3
  go mod tidy
}

BuildReleaseMusl() {
  muslflags="--extldflags '-static -fpic' $ldflags"
  BASE="https://musl.cc/"
  FILES=(x86_64-linux-musl-cross x86_64-w64-mingw32-cross)
  for i in "${FILES[@]}"; do
    url="${BASE}${i}.tgz"
    curl -L -o "${i}.tgz" "${url}"
    sudo tar xf "${i}.tgz" --strip-components 1 -C /usr/local
    rm -f "${i}.tgz"
  done
  OS_ARCHES=(windows-amd64)
  CGO_ARGS=(x86_64-w64-mingw32-gcc)
  for i in "${!OS_ARCHES[@]}"; do
    os_arch=${OS_ARCHES[$i]}
    cgo_cc=${CGO_ARGS[$i]}
    echo building for ${os_arch}
    export GOOS=${os_arch%%-*}
    export GOARCH=${os_arch##*-}
    export CC=${cgo_cc}
    export CGO_ENABLED=1
    go build -o ../build/$appName-$os_arch -ldflags="$muslflags" -tags=jsoniter .
  done
}

BuildReleaseLinuxMuslArm() {
  muslflags="--extldflags '-static -fpic' $ldflags"
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
    export CGO_ENABLED=1
    export GOARM=${arm}
    go build -o ../build/$appName-$os_arch -ldflags="$muslflags" -tags=jsoniter .
  done
}

MakeRelease() {
  cd ../build
  mkdir compress
  for i in $(find . -type f -name "$appName-linux-*"); do
    cp "$i" alist
	upx -9 alist
    tar -czvf compress/"$i".tar.gz alist
  done
  for i in $(find . -type f -name "$appName-windows-*"); do
    cp "$i" alist.exe
	upx -9 alist.exe
    zip compress/"$i".zip alist.exe
  done
  tar -czvf compress/alist.tar.gz ../alist/*
  cd ../
}

if [ "$1" = "release" ]; then
  FetchWebRelease
  PreBuildRelease
  BuildReleaseMusl
  BuildReleaseLinuxMuslArm
  MakeRelease
else
  echo -e "Parameter error"
fi
