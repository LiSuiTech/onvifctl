# ONVIF 命令行工具 (onvifctl)

一个用 Go 语言开发的 ONVIF 协议命令行工具,用于管理和控制支持 ONVIF 的网络摄像头设备。

## 功能特性

- ✅ **智能设备发现**
  - 广播发现 (WS-Discovery)
  - IP 范围扫描
  - 网段扫描 (CIDR)
  - 自动认证检测 (WS-Security/Digest/Basic/None)
  - 显示设备详细信息 (厂商、型号、固件、认证方式)
  - 支持多组凭据自动尝试
  - 结果导出 (文本/JSON)
- ✅ **设备诊断**
  - 连接测试
  - 认证方式测试
  - SOAP 请求/响应查看
  - 端点探测
- ✅ **多种认证方式**
  - WS-UsernameToken (WS-Security) - ONVIF 标准
  - HTTP Digest Authentication - 安全性高
  - HTTP Basic Authentication - 简单兼容
  - 无认证模式 - 开发测试
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

# Windows
go build -o onvifctl.exe

# 安装到系统路径 (可选)
# Linux/macOS
sudo mv onvifctl /usr/local/bin/

# Windows - 添加到 PATH 环境变量
```

## 快速开始

### 1. 发现局域网设备

```bash
# 广播发现 (最快速)
onvifctl discover

# 扫描指定网段
onvifctl discover --mode subnet --subnet 192.168.1.0/24

# 扫描单个 IP
onvifctl discover --mode ip --ip 192.168.1.100
```

### 2. 测试设备连接

```bash
# 测试连接和认证
onvifctl test -H 192.168.1.100 -u admin -w 12345

# 查看详细的 SOAP 交互
onvifctl test -H 192.168.1.100 -u admin -w 12345 --show-soap
```

### 3. 获取设备信息

```bash
onvifctl info -H 192.168.1.100 -u admin -w 12345
```

### 4. 获取视频流

```bash
onvifctl stream -H 192.168.1.100 -u admin -w 12345
```

## 详细使用说明

### 设备发现 (discover)

智能发现局域网内的 ONVIF 设备，支持三种模式。

#### 广播发现模式

使用 WS-Discovery 协议自动发现设备（推荐，最快速）：

```bash
# 基本用法
onvifctl discover

# 指定网卡
onvifctl discover --interface eth0

# 使用自定义凭据
onvifctl discover --cred admin:password --cred root:12345

# 保存结果
onvifctl discover --save devices.txt

# JSON 格式输出
onvifctl discover --json

# 查看详细认证过程
onvifctl discover --verbose
```

#### IP 扫描模式

扫描指定的单个 IP 地址：

```bash
# 扫描单个 IP
onvifctl discover --mode ip --ip 192.168.1.100

# 使用自定义凭据
onvifctl discover --mode ip --ip 192.168.1.100 --cred admin:12345

# 自定义端口
onvifctl discover --mode ip --ip 192.168.1.100 --ports 80,8080,8000

# 调整超时时间
onvifctl discover --mode ip --ip 192.168.1.100 --timeout 5
```

#### 网段扫描模式

扫描指定的 IP 范围或网段：

```bash
# 使用 CIDR 格式
onvifctl discover --mode subnet --subnet 192.168.1.0/24

# 使用 IP 范围
onvifctl discover --mode subnet --start 192.168.1.1 --end 192.168.1.254

# 自定义端口和凭据
onvifctl discover --mode subnet --subnet 192.168.1.0/24 \
  --ports 80,8080 \
  --cred admin:admin \
  --cred admin:12345

# 保存结果
onvifctl discover --mode subnet --subnet 192.168.1.0/24 --save devices.txt
```

**输出示例:**

```
=== 扫描网段: 192.168.1.1 - 192.168.1.254 ===
需要扫描 254 个IP地址, 5 个端口
扫描进度: 5080/5080 (100.0%)

发现 3 个 ONVIF 设备:

序号  IP地址            端口  厂商          型号            固件版本      序列号          认证方式      认证结果
----  ---------------   ----  ----------    ------------    ----------    ------------    ----------    ------------------
1     192.168.1.64      80    Hikvision     DS-2CD2185G0    V5.5.82       DS123456        digest        ✓ 成功(Digest)
2     192.168.1.108     80    Dahua         IPC-HFW1230S    V2.800.0      ABC123XYZ       wsse          ✓ 成功(WS-Security)
3     192.168.1.150     80    -             -               -             -               unknown       ✗ 凭据错误
```

**认证结果说明:**

| 显示 | 含义 |
|------|------|
| `✓ 成功(WS-Security)` | 使用 WS-Security 认证成功 |
| `✓ 成功(Digest)` | 使用 Digest 认证成功 |
| `✓ 成功(Basic)` | 使用 Basic 认证成功 |
| `✓ 成功(无认证)` | 无需认证 |
| `✗ 凭据错误` | 所有凭据都失败 |
| `✗ 失败` | 认证失败 |

#### discover 命令参数

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| --mode | -m | 发现模式: broadcast/ip/subnet | broadcast |
| --interface | -i | 网络接口名称 (broadcast 模式) | 自动 |
| --ip | | 目标 IP 地址 (ip 模式) | - |
| --subnet | | 目标子网 CIDR (如 192.168.1.0/24) | - |
| --start | | 起始 IP 地址 | - |
| --end | | 结束 IP 地址 | - |
| --ports | | 扫描端口列表 | 80,8080,8000,8899,554 |
| --timeout | -t | 连接超时时间(秒) | 2 |
| --cred | -c | 认证凭据 username:password (可多次) | admin:admin等 |
| --save | -o | 保存设备列表到文件 | - |
| --json | | 以 JSON 格式输出 | false |
| --verbose | -v | 显示详细的认证过程 | false |

### 设备测试 (test)

诊断设备连接和认证问题：

```bash
# 基本测试
onvifctl test -H 192.168.1.100 -u admin -w 12345

# 查看 SOAP 请求和响应
onvifctl test -H 192.168.1.100 -u admin -w 12345 --show-soap

# 测试所有认证方式
onvifctl test -H 192.168.1.100 -u admin -w 12345 --test-auth

# 探测可用端点
onvifctl test -H 192.168.1.100 -u admin -w 12345 --test-probe
```

**输出示例:**

```
目标设备: http://192.168.1.100:80/onvif/device_service
用户名: admin
密码: *****

========================================
1. 测试基本连接
========================================
尝试路径: /onvif/device_service
  HTTP 状态: 401 Unauthorized
  ✓ 路径有效
✓ 基本连接成功

========================================
2. 测试所有认证方式
========================================

[1] 测试无认证访问
--------------------
HTTP 状态: 401
✗ 需要认证 (401)

[2] 测试 HTTP Basic 认证
--------------------
HTTP 状态: 401
✗ 认证失败 (401)

[3] 测试 WS-Security 认证
--------------------
HTTP 状态: 200
✓ WS-Security 认证成功

设备信息:
  厂商: Hikvision
  型号: DS-2CD2185G0-I
  固件: V5.5.82
  序列号: DS-2CD2185G0-I20220101AAWRJ12345678
```

#### test 命令参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| --test-auth | 测试所有认证方式 | true |
| --test-probe | 测试设备探测 | false |
| --show-soap | 显示 SOAP 请求和响应 | false |

### 获取设备信息 (info)

```bash
# 基本用法 (HTTP, 默认端口 80)
onvifctl info -H 192.168.1.100 -u admin -w 12345

# 使用 HTTPS (默认端口 443)
onvifctl info -H 192.168.1.100 -P 443 -s -u admin -w 12345

# 使用 HTTPS 和自定义端口
onvifctl info -H 192.168.1.100 -P 8443 -s -u admin -w 12345

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
设备时间:     2025-01-07 08:30:15 UTC
```

### 获取视频流地址 (stream)

```bash
# 获取主码流 (HTTP, 默认端口 80)
onvifctl stream -H 192.168.1.100 -u admin -w 12345

# 使用 HTTPS
onvifctl stream -H 192.168.1.100 -P 443 -s -u admin -w 12345

# 获取子码流 (profile 1)
onvifctl stream -H 192.168.1.100 -u admin -w 12345 -r 1

# 使用 HTTP Digest 认证
onvifctl stream -H 192.168.1.100 -u admin -w 12345 -a digest
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

### PTZ 云台控制 (ptz)

```bash
# 向右移动 (水平速度 0.5)
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --pan 0.5 --timeout 2

# 向左移动
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --pan -0.5 --timeout 2

# 向上移动
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --tilt 0.5 --timeout 2

# 放大 (zoom in)
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action move --zoom 0.5 --timeout 2

# 停止移动
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action stop

# 设置预置位 1
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action setpreset --preset 1

# 转到预置位 1
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action goto --preset 1

# 列出所有预置位
onvifctl ptz -H 192.168.1.100 -u admin -w 12345 --action list
```

### 抓取图像 (snapshot)

```bash
# 抓取主码流图像
onvifctl snapshot -H 192.168.1.100 -u admin -w 12345

# 指定输出文件
onvifctl snapshot -H 192.168.1.100 -u admin -w 12345 -o camera.jpg

# 抓取子码流图像
onvifctl snapshot -H 192.168.1.100 -u admin -w 12345 -r 1 -o substream.jpg
```

### 配置管理 (config)

```bash
# 查看视频编码配置
onvifctl config get-video -H 192.168.1.100 -u admin -w 12345

# 修改视频配置
onvifctl config set-video -H 192.168.1.100 -u admin -w 12345 \
  --width 1920 --height 1080 --fps 25 --bitrate 4096

# 查看网络配置
onvifctl config get-network -H 192.168.1.100 -u admin -w 12345
```

### 时间管理 (time)

```bash
# 获取设备时间
onvifctl time get -H 192.168.1.100 -u admin -w 12345

# 同步设备时间到系统时间
onvifctl time sync -H 192.168.1.100 -u admin -w 12345

# 设置 NTP 服务器
onvifctl time set-ntp -H 192.168.1.100 -u admin -w 12345 --server ntp.aliyun.com
```

### 事件订阅 (events)

```bash
# 订阅所有事件 (持续 60 秒)
onvifctl events -H 192.168.1.100 -u admin -w 12345

# 指定订阅时长
onvifctl events -H 192.168.1.100 -u admin -w 12345 -t 300

# 使用过滤器订阅特定事件
onvifctl events -H 192.168.1.100 -u admin -w 12345 \
  -f "tns1:RuleEngine/CellMotionDetector"
```

### 批量设备管理 (batch)

#### 1. 导出配置模板
```bash
onvifctl batch export --file devices.yaml
```

#### 2. 编辑配置文件
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

## 全局参数

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| --host | -H | 设备 IP 地址或主机名 | 必填 |
| --port | -P | 设备端口号 | 80 |
| --user | -u | ONVIF 登录用户名 | admin |
| --pass | -w | ONVIF 登录密码 | 必填 |
| --auth | -a | 认证模式 (ws-security/digest) | ws-security |
| --https | -s | 使用 HTTPS 协议 | false |
| --debug | -d | 启用调试日志 | false |

## 使用场景

### 场景 1: 新环境部署

```bash
# 1. 发现所有设备
onvifctl discover --mode subnet --subnet 192.168.1.0/24 \
  --cred admin:admin --cred admin:12345 \
  --save discovered.txt

# 2. 测试特定设备
onvifctl test -H 192.168.1.100 -u admin -w 12345

# 3. 获取详细信息
onvifctl info -H 192.168.1.100 -u admin -w 12345

# 4. 获取视频流
onvifctl stream -H 192.168.1.100 -u admin -w 12345
```

### 场景 2: 故障排查

```bash
# 1. 测试连接和认证
onvifctl test -H 192.168.1.100 -u admin -w 12345 --show-soap

# 2. 尝试不同认证方式
onvifctl info -H 192.168.1.100 -u admin -w 12345 -a digest
onvifctl info -H 192.168.1.100 -u admin -w 12345 -a ws-security

# 3. 查看详细日志
onvifctl info -H 192.168.1.100 -u admin -w 12345 -d
```

### 场景 3: 批量设备管理

```bash
# 1. 发现并导出
onvifctl discover --mode subnet --subnet 192.168.1.0/24 \
  --json > devices.json

# 2. 创建批量配置
onvifctl batch export --file devices.yaml
# 编辑 devices.yaml

# 3. 批量操作
onvifctl batch info --file devices.yaml
onvifctl batch snapshot --file devices.yaml
onvifctl batch sync-time --file devices.yaml
```

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

## 技术实现

### ONVIF 认证方式

#### 1. WS-Security UsernameToken (默认, 推荐)
- ONVIF 标准认证方式
- 密码加密传输 (SHA1 + Base64)
- 包含时间戳防止重放攻击
- 公式: PasswordDigest = Base64(SHA1(Nonce + Created + Password))

#### 2. HTTP Digest Authentication
- HTTP 标准认证
- 密码不明文传输
- Challenge-Response 机制
- 响应计算: MD5(hash1:nonce:nc:cnonce:qop:hash2)

#### 3. HTTP Basic Authentication
- 最简单的认证方式
- Base64 编码 (非加密)
- 仅在 HTTPS 下安全
- 兼容性好

#### 4. 无认证模式
- 开发测试用
- 设备默认开放
- 不推荐生产环境

### 支持的 ONVIF 服务

**设备服务 (Device Service):**
- GetDeviceInformation - 获取设备信息
- GetSystemDateAndTime - 获取系统时间
- SetSystemDateAndTime - 设置系统时间
- GetCapabilities - 获取设备能力
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

**发现服务 (Discovery):**
- WS-Discovery Probe - 广播发现
- IP 范围扫描 - 主动探测
- 多种认证方式自动检测

## 常见问题

### 1. 设备发现找不到设备

**可能原因:**
- 防火墙阻止了 UDP 3702 端口
- 设备未启用 ONVIF
- 不在同一网段

**解决方法:**
```bash
# 尝试 IP 扫描模式
onvifctl discover --mode subnet --subnet 192.168.1.0/24

# 指定网卡
onvifctl discover --interface eth0

# 测试单个设备
onvifctl test -H 192.168.1.100 -u admin -w 12345
```

### 2. 认证失败

**可能原因:**
- 用户名或密码错误
- 认证方式不匹配
- ONVIF 用户权限不足

**解决方法:**
```bash
# 使用 test 命令诊断
onvifctl test -H 192.168.1.100 -u admin -w 12345 --show-soap

# 尝试不同认证方式
onvifctl info -H 192.168.1.100 -u admin -w 12345 -a digest
onvifctl info -H 192.168.1.100 -u admin -w 12345 -a ws-security

# 尝试常见密码
onvifctl discover --mode ip --ip 192.168.1.100 \
  --cred admin:admin \
  --cred admin:12345 \
  --cred admin:admin123
```

### 3. 无法获取流地址

**可能原因:**
- 设备不支持 RTSP
- Profile 索引错误
- 视频编码未配置

**解决方法:**
```bash
# 查看所有 Profile
onvifctl stream -H 192.168.1.100 -u admin -w 12345

# 尝试不同索引
onvifctl stream -H 192.168.1.100 -u admin -w 12345 -r 0
onvifctl stream -H 192.168.1.100 -u admin -w 12345 -r 1

# 查看视频编码配置
onvifctl config get-video -H 192.168.1.100 -u admin -w 12345
```

### 4. 扫描速度慢

**优化方法:**
```bash
# 减少扫描端口
onvifctl discover --mode subnet --subnet 192.168.1.0/24 --ports 80,8080

# 缩短超时时间
onvifctl discover --mode subnet --subnet 192.168.1.0/24 --timeout 1

# 缩小扫描范围
onvifctl discover --mode subnet --start 192.168.1.100 --end 192.168.1.150
```

## 项目结构

```
onvifctl/
├── main.go                      # 主程序和命令行接口
├── discover_cmd.go              # 设备发现命令实现
├── test_onvif.go               # 设备测试和诊断工具
├── discovery/
│   ├── discovery.go            # 设备发现核心逻辑
│   ├── device_info.go          # 设备信息管理
│   └── device_info_enhanced.go # 增强的认证处理
├── go.mod                       # Go 模块配置
├── go.sum                       # 依赖校验
└── README.md                    # 说明文档
```

## 开发计划

- [x] 基础设备信息获取
- [x] RTSP 流地址获取
- [x] 多种认证方式支持
- [x] PTZ 云台控制
- [x] 图像抓取
- [x] 配置管理
- [x] 智能设备发现 (广播/IP/网段)
- [x] 自动认证检测
- [x] 设备诊断工具
- [x] 时间管理
- [x] 批量设备管理
- [x] 结果导出 (文本/JSON)
- [ ] 音频配置
- [ ] 用户管理
- [ ] 完整的事件处理
- [ ] 设备备份/恢复
- [ ] Web 管理界面

## 常见设备默认凭据

| 厂商 | 默认用户名 | 默认密码 | 认证方式 |
|------|-----------|---------|---------|
| Hikvision | admin | 需激活设置 | Digest/WS-Security |
| Dahua | admin | admin | WS-Security |
| Uniview | admin | 123456 | WS-Security |
| TP-Link | admin | admin | Basic/Digest |
| Axis | root | pass | Digest |

**注意:** 生产环境务必修改默认密码！

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request!

## 更新日志

### v1.1.0 (2025-01-07)
- ✨ 新增智能设备发现功能
  - 支持广播发现、IP 扫描、网段扫描三种模式
  - 自动检测认证方式 (WS-Security/Digest/Basic/None)
  - 显示设备详细信息和认证结果
  - 支持多组凭据自动尝试
- ✨
