# ONVIF 命令行工具 (onvifctl)

一个用 Go 语言开发的 ONVIF 协议命令行工具,用于管理和控制支持 ONVIF 的网络摄像头设备。

## 功能特性

- ✅ **设备发现**
  - WS-Discovery 协议自动发现局域网设备
  - 支持指定网络接口
  - 导出设备列表
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
- ✅ **时间管理**
  - 获取设备时间
  - 同步到系统时间
  - 设置 NTP 服务器
- ✅ **事件订阅**
  - 监听设备事件
  - 移动侦测、报警等
- ✅ **批量设备管理**
  - 配置文件导入/导出
  - 批量获取信息
  - 批量抓图
  - 批量时间同步
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
onvifctl stream -H 192.168.1.100 -u admin -w 12345 -p 1

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
onvifctl snapshot -H 192.168.1.100 -u admin -w 12345 -p 1 -o substream.jpg
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

### 设备发现

```bash
# 自动发现局域网内的 ONVIF 设备
onvifctl discover

# 指定超时时间 (默认 5 秒)
onvifctl discover -t 10

# 指定网络接口
onvifctl discover -i eth0

# 保存发现的设备到文件
onvifctl discover -o discovered.yaml

# 启用调试模式查看详细信息
onvifctl discover -d
```

**输出示例:**
```
发现 3 个设备:

设备 1:
  地址: urn:uuid:4d454930-0023-1002-8000-a0369f123456
  XAddrs: http://192.168.1.100/onvif/device_service
  类型: dn:NetworkVideoTransmitter
  范围: onvif://www.onvif.org/name/Hikvision

设备 2:
  地址: urn:uuid:5e565041-0034-2103-9111-b1479f234567
  XAddrs: http://192.168.1.101/onvif/device_service
  类型: dn:NetworkVideoTransmitter
  范围: onvif://www.onvif.org/name/Dahua

设备 3:
  地址: urn:uuid:6f676152-0045-3204-a222-c2589f345678
  XAddrs: http://192.168.1.102/onvif/device_service
  类型: dn:NetworkVideoTransmitter
  范围: onvif://www.onvif.org/name/Uniview
```

### 时间管理

```bash
# 获取设备时间
onvifctl time get -H 192.168.1.100 -u admin -w 12345

# 同步设备时间到系统时间
onvifctl time sync -H 192.168.1.100 -u admin -w 12345

# 设置 NTP 服务器
onvifctl time set-ntp -H 192.168.1.100 -u admin -w 12345 --server ntp.aliyun.com
onvifctl time set-ntp -H 192.168.1.100 -u admin -w 12345 --server pool.ntp.org
```

**输出示例:**
```
=== 设备时间 ===
时间类型: NTP
设备时间: 2025-12-04 03:15:30 UTC
时间差异: 120 秒

✓ 时间同步成功
  设备时间已设置为: 2025-12-04 03:17:30 UTC
```

### 事件订阅

```bash
# 订阅所有事件 (持续 60 秒)
onvifctl events -H 192.168.1.100 -u admin -w 12345

# 指定订阅时长
onvifctl events -H 192.168.1.100 -u admin -w 12345 -t 300

# 使用过滤器订阅特定事件
onvifctl events -H 192.168.1.100 -u admin -w 12345 -f "tns1:RuleEngine/CellMotionDetector"
```

### 批量设备管理

#### 1. 导出配置模板
```bash
# 生成配置文件模板
onvifctl batch export --file devices.yaml
```

#### 2. 编辑配置文件
编辑 `devices.yaml` 添加你的设备信息：
```yaml
devices:
  - name: "Camera-Entrance"
    host: "192.168.1.100"
    port: 80
    username: "admin"
    password: "12345"
    use_https: false
  
  - name: "Camera-Parking"
    host: "192.168.1.101"
    port: 80
    username: "admin"
    password: "12345"
    use_https: false
```

#### 3. 批量操作
```bash
# 导入并验证配置
onvifctl batch import --file devices.yaml

# 批量获取所有设备信息
onvifctl batch info --file devices.yaml

# 批量抓取所有设备图像
onvifctl batch snapshot --file devices.yaml --output snapshots

# 批量同步所有设备时间
onvifctl batch sync-time --file devices.yaml
```

**输出示例:**
```
正在获取 4 个设备的信息...

[1] Camera-Entrance - Hikvision DS-2CD2143G0-I (固件: V5.7.3)
[2] Camera-Parking - Hikvision DS-2CD2043G0-I (固件: V5.7.3)
[3] Camera-Hallway - Dahua IPC-HFW2431S (固件: V2.800.0000000.25.R)
[4] Camera-BackDoor - Uniview IPC322SR3-DVS28 (固件: V3.0.0.8)

✓ 批量查询完成
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

#### discover 命令

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| --timeout | -t | 发现超时时间(秒) | 5 |
| --interface | -i | 网络接口名称 | 自动 |
| --save | -o | 保存设备列表到文件 | - |

#### time 子命令

| 子命令 | 说明 |
|--------|------|
| get | 获取设备时间 |
| sync | 同步设备时间到系统时间 |
| set-ntp | 设置 NTP 服务器 |

**set-ntp 参数:**
| 参数 | 说明 |
|------|------|
| --server | NTP 服务器地址 |

#### events 命令

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| --duration | -t | 订阅持续时间(秒) | 60 |
| --filter | -f | 事件过滤器 | - |

#### batch 子命令

| 子命令 | 说明 |
|--------|------|
| import | 从配置文件导入设备列表 |
| export | 导出设备配置到文件 |
| info | 批量获取设备信息 |
| snapshot | 批量抓取图像 |
| sync-time | 批量同步设备时间 |

**公共参数:**
| 参数 | 说明 | 默认值 |
|------|------|--------|
| --file | 配置文件路径 | devices.yaml |
| --output | 输出目录 (snapshot) | snapshots |

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
├── advanced.go     # 高级功能(发现、事件、批量管理)
├── go.mod          # Go 模块配置
├── devices.yaml    # 批量设备配置文件
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
- SetSystemDateAndTime - 设置系统时间
- SetNTP - 设置 NTP 服务器
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

**事件服务 (Event Service):**
- Subscribe - 订阅事件
- CreatePullPointSubscription - 创建拉取点订阅
- PullMessages - 拉取消息

**发现服务 (Discovery):**
- WS-Discovery Probe - 设备发现

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
- [x] 设备发现 (WS-Discovery)
- [x] 时间同步
- [x] 批量设备管理
- [x] 配置导入/导出
- [x] 事件订阅 (框架)
- [ ] 音频配置
- [ ] 用户管理
- [ ] 完整的事件处理 (PullPoint)
- [ ] 设备备份/恢复
- [ ] Web 管理界面
- [ ] 日志记录和审计

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request!
