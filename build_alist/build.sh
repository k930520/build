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
  sed -i '/^  "AliyundriveShare".*: {$/r drivers.txt' src/lang/zh-CN/drivers.json
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
  
 #  sed -i '/stdpath "path"/a\
 #  	"regexp"\
	# "strings"\
 #  ' internal/op/fs.go
  
 #  sed -i '/files, err := storage.List(ctx, dir, args)/i\
	# 	args.ReqPath = key\
 #  ' internal/op/fs.go
  
 #  sed -i '/log.Debugf("set cache: \%s => \%+v", key, files)/i\
	# 			reqPath := strings.Split(args.ReqPath, "/")\
	# 			if storage.GetStorage().Driver == "AliyundriveShare" && (regexp.MustCompile(`^[A-Z0-9]`).MatchString(reqPath[len(reqPath)-1]) || regexp.MustCompile(`^[A-Z0-9]`).MatchString(reqPath[len(reqPath)-2])) {\
	# 				log.Debugf("set cache: %s => %+v", key, files)\
	# 				listCache.Set(key, files, cache.WithEx[[]model.Obj](time.Minute*1))\
	# 			} else {\
 #  ' internal/op/fs.go
  
 #  sed -i '/listCache\.Set(key, files, cache\.WithEx\[\[\]model.Obj\](time\.Minute\*time\.Duration(storage\.GetStorage()\.CacheExpiration)))/a\
	# 			}\
 #  ' internal/op/fs.go

 #  sed -i '/stdpath "path"/a\
	# "os"\
	# "github.com/alist-org/alist/v3/internal/offline_download/aria2"\
	# "github.com/alist-org/alist/v3/internal/offline_download/tool"\
 #  ' internal/fs/copy.go

 #  sed -i -e '/return errors\.WithMessagef(err, "failed get \[\%s\] link", srcFilePath)/{n;d;}' internal/fs/copy.go

 #  sed -i -e '/return errors\.WithMessagef(err, "failed get \[\%s\] link", srcFilePath)/a\
	# }\
	# _aria2 := aria2.Aria2{}\
	# _, err = _aria2.Init()\
	# if err != nil {\
 #  ' internal/fs/copy.go

 #  sed -i -e '/return op\.Put(tsk\.Ctx(), dstStorage, dstDirPath, ss, tsk\.SetProgress, true)/a\
	# } else {\
	# 	gid, err := _aria2.AddURL(&tool.AddUrlArgs{\
	# 		Url:     link.URL,\
	# 		Out:     stdpath.Base(srcFilePath),\
	# 		TempDir: conf.Conf.TempDir,\
	# 	})\
	# 	if err != nil {\
	# 		return err\
	# 	}\
	# 	for {\
	# 		status, err := _aria2.Status(&tool.DownloadTask{GID: gid})\
	# 		if err != nil {\
	# 			return err\
	# 		}\
	# 		if status.Completed {\
	# 			tempFile := stdpath.Join(conf.Conf.TempDir, stdpath.Base(srcFilePath))\
	# 			rc, err := os.Open(tempFile)\
	# 			if err != nil {\
	# 				return errors.Wrapf(err, "failed to open file %s")\
	# 			}\
	# 			rcInfo, err := rc.Stat()\
	# 			if err != nil {\
	# 				return errors.Wrapf(err, "failed get file info %s")\
	# 			}\
	# 			ss := &stream.FileStream{\
	# 				Ctx: tsk.Ctx(),\
	# 				Obj: &model.Object{\
	# 					Name:     stdpath.Base(srcFilePath),\
	# 					Size:     rcInfo.Size(),\
	# 					Modified: rcInfo.ModTime(),\
	# 					IsFolder: false,\
	# 				},\
	# 				Mimetype: utils.GetMimeType(srcFilePath),\
	# 				Reader:   rc,\
	# 				Closers:  utils.NewClosers(rc),\
	# 			}\
	# 			err = op.Put(tsk.Ctx(), dstStorage, dstDirPath, ss, tsk.SetProgress)\
	# 			if err == nil {\
	# 				os.Remove(tempFile)\
	# 			}\
	# 			return err\
	# 		}\
	# 		time.Sleep(time.Second * 3)\
	# 	}\
	# }\
 #  ' internal/fs/copy.go

 #  sed -i '/"dir": args\.TempDir,/a\
	# 	"out": args.Out,\
 #  ' internal/offline_download/aria2/aria2.go

 #  sed -i '/UID     string/a\
	# Out     string\
 #  ' internal/offline_download/tool/base.go
  
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
