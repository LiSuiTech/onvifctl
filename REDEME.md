# ONVIF 命令行工具 (onvifctl)

一个用 Go 语言开发的 ONVIF 协议命令行工具,用于管理和控制支持 ONVIF 的网络摄像头设备。

## 功能特性

- ✅ **多种认证方式**
  - WS-UsernameToken (Digest) - 默认,所有 ONVIF 设备必须支持
  - HTTP Digest Authentication - RTSP/HTTP 标准认证
- ✅ **支持 HTTP 和 HTTPS**
  - HTTP - 默认,端口 80
  - HTTPS/TLS - 加密通信,端口 443
- ✅ **PTZ 云台控制**
  - 连续移动 (水平/垂直/缩放)
  - 停止移动
  - 预置位管理 (设置/转到/列表)
- ✅ **图像抓取**
  - 从实时流抓取 JPEG 图像
  - 支持不同配置文件
- ✅ **配置管理**
  - 查看视频编码配置
  - 修改分辨率、帧率、比特率
  - 查看网络配置
- ✅ 获取设备信息 (厂商、型号、固件版本等)
- ✅ 获取系统时间
- ✅ 获取 RTSP 视频流地址
- ✅ 支持多配置文件 (Profile) 选择
- ✅ 调试模式查看 SOAP 请求/响应

## 安装

### 前置要求

- Go 1.21 或更高版本

### 编译安装

```bash
# 克隆代码
git clone <your-repo-url>
cd onvifctl

# 下载依赖
go mod download

# 编译
go build -o onvifctl

# 安装到系统路径 (可选)
sudo mv onvifctl /usr/local/bin/
```

## 使用方法

### 查看帮助

```bash
onvifctl -h
```

### 获取设备信息

```bash
# 基本用法 (HTTP, 默认端口 80)
onvifctl info -H 192.168.1.100 -u admin -w 12345

# 使用 HTTPS (默认端口 443)
onvifctl info -H 192.168.1.100 -P 443 -s -u admin -w 12345

# 使用 HTTPS 和自定义端口
onvifctl info -H 192.168.1.100 -P 8443 -s -u admin -w 12345

# 指定端口 (HTTP)
onvifctl info -H 192.168.1.100 -P 8080 -u admin -w 12345

# 使用 HTTP Digest 认证
onvifctl info -H 192.168.1.100 -u admin -w 12345 -a digest

# 启用调试模式
onvifctl info -H 192.168.1.100 -u admin -w 12345 -d
```

**输出示例:**
```
=== 设备信息 ===
制造商:       Hikvision
型号:         DS-2CD2143G0-I
固件版本:     V5.7.3
序列号:       DS-2CD2143G0-I20220101AAWRJ12345678
硬件 ID:      88888888
时间类型:     NTP
设备时间:     2025-12-03 08:30:15 UTC
```

### 获取视频流地址

```bash
# 获取主码流 (HTTP, 默认端口 80)
onvifctl stream -H 192.168.1.100 -u admin -w 12345

# 使用 HTTPS
onvifctl stream -H 192.168.1.100 -P 443 -s -u admin -w 12345

# 指定端口号
onvifctl stream -H 192.168.1.100 -P 8080 -u admin -w 12345

# 获取子码流 (profile 1)
onvifctl stream -H 192.168.1.100 -u admin -w 12345 -r 1

# 使用 HTTP Digest 认证
onvifctl stream -H 192.168.1.100 -u admin -w 12345 -a digest

# HTTPS + Digest 认证
onvifctl stream -H 192.168.1.100 -P 443 -s -u admin -w 12345 -a digest
```

**输出示例:**
```
=== 视频流信息 ===
配置名称:     MainStream
配置 Token:   Profile_1
RTSP 地址:    rtsp://192.168.1.100:554/Streaming/Channels/101

所有可用配置:
  [0] MainStream (Token: Profile_1)
  [1] SubStream (Token: Profile_2)
```

### PTZ 云台控制

```bash
# 向右移动 (水平速度 0.5)
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --pan 0.5 --timeout 2

# 向左移动
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --pan -0.5 --timeout 2

# 向上移动
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --tilt 0.5 --timeout 2

# 向下移动
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --tilt -0.5 --timeout 2

# 放大 (zoom in)
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --zoom 0.5 --timeout 2

# 缩小 (zoom out)
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --zoom -0.5 --timeout 2

# 停止移动
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action stop

# 设置预置位 1
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action setpreset --preset 1

# 转到预置位 1
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action goto --preset 1

# 列出所有预置位
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action list
```

**输出示例:**
```
✓ PTZ 移动命令已发送
  水平速度: 0.50
  垂直速度: 0.00
  缩放速度: 0.00
  持续时间: 2 秒
```

### 抓取图像

```bash
# 抓取主码流图像
onvifctl snapshot -H 192.168.1.100 -u admin -w 12345

# 指定输出文件
onvifctl snapshot -H 192.168.1.100 -u admin -w 12345 -o camera.jpg

# 抓取子码流图像
onvifctl snapshot -H 192.168.1.100 -u admin -w 12345 -r 1 -o substream.jpg
```

**输出示例:**
```
✓ 图像已保存到: snapshot.jpg
  配置: MainStream
  大小: 245678 字节
```

### 配置管理

```bash
# 查看视频编码配置
onvifctl config get-video -H 192.168.1.100 -u admin -w 12345

# 修改视频配置 (分辨率 1920x1080, 25fps, 4096kbps)
onvifctl config set-video -H 192.168.1.100 -u admin -w 12345 \
  --width 1920 --height 1080 --fps 25 --bitrate 4096

# 查看网络配置
onvifctl config get-network -H 192.168.1.100 -u admin -w 12345
```

**输出示例:**
```
=== 视频编码配置 ===

配置 0:
  Token:      VideoEncoderToken
  名称:       MainStream
  编码:       H264
  分辨率:     1920x1080
  质量:       5
  帧率:       25 fps
  比特率:     4096 kbps
```

### 命令行参数

#### 全局参数

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| --host | -H | 设备 IP 地址或主机名 | 必填 |
| --port | -P | 设备端口号 | 80 |
| --user | -u | ONVIF 登录用户名 | admin |
| --pass | -w | ONVIF 登录密码 | 必填 |
| --auth | -a | 认证模式 (ws-security/digest) | ws-security |
| --https | -s | 使用 HTTPS 协议 | false |
| --debug | -d | 启用调试日志 | false |

#### info 命令

无额外参数,使用全局参数即可。

#### stream 命令

| 参数 | 简写  | 说明 | 默认值 |
|------|-----|------|--------|
| --profile | -r  | 配置文件索引 | 0 |

#### ptz 命令

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| --action | | 操作类型 (必填) | - |
| --pan | | 水平速度 (-1.0 到 1.0) | 0 |
| --tilt | | 垂直速度 (-1.0 到 1.0) | 0 |
| --zoom | | 缩放速度 (-1.0 到 1.0) | 0 |
| --timeout | | 移动持续时间(秒) | 1 |
| --preset | | 预置位编号 | 0 |

**action 支持的值:**
- `move` - 连续移动
- `stop` - 停止移动
- `goto` - 转到预置位
- `setpreset` - 设置预置位
- `list` - 列出所有预置位

#### snapshot 命令

| 参数 | 简写  | 说明 | 默认值 |
|------|-----|------|--------|
| --output | -o  | 输出文件路径 | snapshot.jpg |
| --profile | -r  | 配置文件索引 | 0 |

#### config 子命令

| 子命令 | 说明 |
|--------|------|
| get-video | 获取视频编码配置 |
| set-video | 设置视频编码配置 |
| get-network | 获取网络配置 |

**set-video 参数:**
| 参数 | 说明 |
|------|------|
| --width | 视频宽度 |
| --height | 视频高度 |
| --fps | 帧率 |
| --bitrate | 比特率 (kbps) |

## 使用 RTSP 流

获取到 RTSP 地址后,可以使用以下工具播放:

### 使用 ffplay

```bash
ffplay "rtsp://admin:12345@192.168.1.100:554/Streaming/Channels/101"
```

### 使用 VLC

```bash
vlc "rtsp://admin:12345@192.168.1.100:554/Streaming/Channels/101"
```

### 使用 ffmpeg 录制

```bash
ffmpeg -i "rtsp://admin:12345@192.168.1.100:554/Streaming/Channels/101" \
       -c copy -t 60 output.mp4
```

## 项目结构

```
onvifctl/
├── main.go         # 主程序和命令行接口
├── client.go       # ONVIF 客户端实现
├── go.mod          # Go 模块配置
└── README.md       # 说明文档
```

## 技术实现

### 协议支持

#### HTTP
- 标准 HTTP/1.1 协议
- 默认端口 80
- 适用于内网环境

#### HTTPS/TLS
- 支持 TLS 加密通信
- 默认端口 443
- 跳过证书验证（开发/测试环境）
- 生产环境建议添加证书验证

### ONVIF 认证方式

#### 1. WS-Security UsernameToken (默认)
使用 WS-Security UsernameToken Profile 1.0 进行认证:
- PasswordDigest = Base64(SHA1(Nonce + Created + Password))
- 每次请求生成新的 Nonce 和时间戳
- 在 SOAP 消息头中传递认证信息

#### 2. HTTP Digest Authentication
标准的 HTTP Digest 认证流程:
- 首次请求返回 401 + WWW-Authenticate 头
- 计算响应: MD5(hash1:nonce:nc:cnonce:qop:hash2)
- hash1 = MD5(username:realm:password)
- hash2 = MD5(POST:uri)
- 在 HTTP Authorization 头中传递认证信息

### SOAP 通信

- 使用 SOAP 1.2 协议
- 支持设备服务 (device_service)
- 支持媒体服务 (media_service)

### 支持的 ONVIF 服务

**设备服务 (Device Service):**
- GetDeviceInformation - 获取设备信息
- GetSystemDateAndTime - 获取系统时间
- GetNetworkInterfaces - 获取网络配置

**媒体服务 (Media Service):**
- GetProfiles - 获取媒体配置
- GetStreamUri - 获取流地址
- GetSnapshotUri - 获取抓图地址
- GetVideoEncoderConfigurations - 获取视频编码配置
- SetVideoEncoderConfiguration - 设置视频编码配置

**PTZ 服务 (PTZ Service):**
- ContinuousMove - 连续移动
- Stop - 停止移动
- GotoPreset - 转到预置位
- SetPreset - 设置预置位
- GetPresets - 获取预置位列表

## 常见问题

### 1. 连接设备失败

- 确认设备 IP 地址正确
- 确认端口号正确（HTTP: 80/8080, HTTPS: 443/8443）
- 确认设备支持 ONVIF 协议
- 检查网络连接是否正常
- 如果使用 HTTPS,确认设备已启用 TLS
- 尝试使用 `-d` 参数查看详细日志

### 2. 认证失败

- 确认用户名和密码正确
- 某些设备可能需要先在设备管理界面启用 ONVIF
- 检查设备的 ONVIF 用户权限设置
- 尝试切换认证模式: `-a digest` 或 `-a ws-security`
- 使用 `-d` 参数查看详细的认证过程

### 3. 无法获取流地址

- 确认设备支持 RTSP 协议
- 尝试不同的 profile 索引 (`-r 0`, `-r 1` 等)
- 检查设备的视频编码设置

## 开发计划

- [x] 设备信息获取
- [x] RTSP 流地址获取
- [x] PTZ 云台控制
- [x] 图像抓取
- [x] 视频编码配置
- [x] 网络配置查看
- [ ] 设备发现 (WS-Discovery)
- [ ] 事件订阅
- [ ] 音频配置
- [ ] 用户管理
- [ ] 时间同步
- [ ] 批量设备管理
- [ ] 配置导入/导出

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request!