# Xray 编译 Android aar

Xray 在实现上使用 go 语言，同时客户端和服务端共用同一套代码，根据配置文件的不同来判断是客户端还是服务端。如果想在 Android 的客户端上运行 V2Ray，那么就需要知道如何编译 v2ray-core 到 aar 库，提供给 Android 客户端。

## 1. 未解决问题
### 1.1. gomobile 不支持 Go module 编译
如果在使用 gomobile 编译时，使用 Go module(也就是 GO111MODULE=on ) 会报错，找不到 package, 参考 https://github.com/golang/go/wiki/Mobile  
![](https://github.com/fivetime/AndroidLibXrayLite/raw/master/screenshot/gomobile.jpg)

## 2. 提前准备
因为编译需要使用到 tun2socks 组件，所以建议先阅读一下相关的知识点。  
http://arloor.com/posts/other/android-vpnservice-and-vpn-dev/  
https://briteming.blogspot.com/2017/09/socksvpn-fqrouter.html  

## 3. 配置环境
### 3.1. Mac OS
#### 3.1.1. Install android-sdk && android-ndk 环境
最简单的办法，直接安装 Android Studio，然后在 Android Studio 中配置:
![](https://github.com/fivetime/AndroidLibXrayLite/raw/master/screenshot/androidsdk1.jpg)  
![](https://github.com/fivetime/AndroidLibXrayLite/raw/master/screenshot/androidsdk2.jpg)  
其中，SDK Platforms 下载几个 Android 的版本。  

#### 3.1.2. 安装 GO
go 我使用的版本: go version go1.13.1 darwin/amd64

### 3.1.3. clone 仓库
```
# clone 仓库到本地
git clone https://github.com/fivetime/AndroidLibXrayLite.git
```

#### 3.1.4. 执行命令
```
###### 使用梯子
export ALL_PROXY=socks5://127.0.0.1:1086
export all_proxy=socks5://127.0.0.1:1086

###### 配置 android 的 sdk 和 ndk
export ANDROID_HOME="/Users/MoMo/Library/Android/sdk"
export ANDROID_NDK_HOME="/Users/MoMo/Library/Android/sdk/ndk/20.0.5594570"

###### 根据环境选择执行脚本:
# Mac OS
sh build/build-on-mac.sh

# 编译 android
sh build-on-mac.sh android [data] [dep]
### data 参数表示会更新 geoip.dat  geosite.dat 文件
### dep 表示会执行 go get -u 更新本地的依赖
```

### 3.2. Linux(Ubuntu/Debian/CentOS)
#### 3.2.1. clone 仓库
```
# clone 仓库到本地
git clone https://github.com/fivetime/AndroidLibXrayLite.git
cd AndroidLibXrayLite/

# 执行脚本
/bin/bash build/build-on-linux.sh sdk data dep
### sdk 表示会安装 Android SDK Tool 和 NDK
### data 参数表示会更新 geoip.dat  geosite.dat 文件
### dep 表示会执行 go get -u 更新本地的依赖
```

### 3.2.2. 编译过程
1. 安装 Docker
1. 安装 Go
1. 安装 Android SDK + NDK + Tools + OpenJDK8
1. Docker 拉取 fivetime/android-v2ray-build:1.0.0 镜像后开始编译

### 3.3. 注意事项
1. 注意需要梯子
1. 脚本默认 GOATH = ~/go 目录
1. 会自动下载 geoip.data 和 geosite.dat 文件
1. 编译生成的 aar 和 source 在 ${GOPATH}/src/AndroidLibXrayLite 目录下

### Mac OS :
![](https://github.com/fivetime/AndroidLibXrayLite/raw/master/screenshot/macosdir.jpg)
