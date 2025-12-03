package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ONVIFClient struct {
	Host       string
	Port       int
	Username   string
	Password   string
	Debug      bool
	UseHTTPS   bool
	XAddr      string
	MediaAddr  string
	AuthMode   string // "ws-security" 或 "digest"
	nc         int    // digest 认证计数器
	httpClient *http.Client
}

// 创建 ONVIF 客户端
func NewONVIFClient(host string, port int, username, password string, debug, useHTTPS bool) (*ONVIFClient, error) {
	// 确定协议
	protocol := "http"
	if useHTTPS {
		protocol = "https"
	}

	// 构建带端口的地址
	hostWithPort := fmt.Sprintf("%s:%d", host, port)

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 如果使用 HTTPS,配置 TLS
	if useHTTPS {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 跳过证书验证（生产环境应该验证证书）
			},
		}
	}

	client := &ONVIFClient{
		Host:       host,
		Port:       port,
		Username:   username,
		Password:   password,
		Debug:      debug,
		UseHTTPS:   useHTTPS,
		XAddr:      fmt.Sprintf("%s://%s/onvif/device_service", protocol, hostWithPort),
		MediaAddr:  fmt.Sprintf("%s://%s/onvif/media_service", protocol, hostWithPort),
		AuthMode:   "ws-security", // 默认使用 WS-Security
		nc:         0,
		httpClient: httpClient,
	}

	if debug {
		fmt.Printf("连接到设备: %s:%d\n", host, port)
		fmt.Printf("协议: %s\n", protocol)
		fmt.Printf("设备服务地址: %s\n", client.XAddr)
		fmt.Printf("媒体服务地址: %s\n", client.MediaAddr)
		fmt.Printf("认证模式: %s\n", client.AuthMode)
		if useHTTPS {
			fmt.Println("⚠️  HTTPS 已启用 (跳过证书验证)")
		}
	}

	return client, nil
}

// 生成 WS-Security 认证头
func (c *ONVIFClient) generateAuth() *Header {
	nonce := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	created := time.Now().UTC().Format(time.RFC3339)

	nonceB64 := base64.StdEncoding.EncodeToString(nonce)

	h := sha1.New()
	h.Write(nonce)
	h.Write([]byte(created))
	h.Write([]byte(c.Password))
	digest := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return &Header{
		Security: Security{
			MustUnderstand: "1",
			UsernameToken: UsernameToken{
				Username: c.Username,
				Password: Password{
					Type:  "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest",
					Value: digest,
				},
				Nonce:   nonceB64,
				Created: created,
			},
		},
	}
}

// 发送 SOAP 请求
func (c *ONVIFClient) sendRequest(url string, request interface{}) ([]byte, error) {
	env := Envelope{
		Header: c.generateAuth(),
		Body: Body{
			Content: request,
		},
	}

	xmlData, err := xml.MarshalIndent(env, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	if c.Debug {
		fmt.Printf("\n请求:\n%s\n", string(xmlData))
	}

	resp, err := http.Post(url, "application/soap+xml", bytes.NewReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if c.Debug {
		fmt.Printf("\n响应:\n%s\n", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	return body, nil
}

// 获取设备信息
func (c *ONVIFClient) GetDeviceInfo() error {
	// 获取设备基本信息
	respData, err := c.sendRequest(c.XAddr, &GetDeviceInformation{})
	if err != nil {
		return err
	}

	var infoResp struct {
		Body struct {
			GetDeviceInformationResponse GetDeviceInformationResponse
		}
	}

	if err := xml.Unmarshal(respData, &infoResp); err != nil {
		return fmt.Errorf("解析设备信息失败: %w", err)
	}

	info := infoResp.Body.GetDeviceInformationResponse

	// 获取系统时间
	timeData, err := c.sendRequest(c.XAddr, &GetSystemDateAndTime{})
	if err == nil {
		var timeResp struct {
			Body struct {
				GetSystemDateAndTimeResponse GetSystemDateAndTimeResponse
			}
		}
		xml.Unmarshal(timeData, &timeResp)

		dt := timeResp.Body.GetSystemDateAndTimeResponse.SystemDateAndTime

		fmt.Println("=== 设备信息 ===")
		fmt.Printf("制造商:       %s\n", info.Manufacturer)
		fmt.Printf("型号:         %s\n", info.Model)
		fmt.Printf("固件版本:     %s\n", info.FirmwareVersion)
		fmt.Printf("序列号:       %s\n", info.SerialNumber)
		fmt.Printf("硬件 ID:      %s\n", info.HardwareId)
		fmt.Printf("时间类型:     %s\n", dt.DateTimeType)
		fmt.Printf("设备时间:     %04d-%02d-%02d %02d:%02d:%02d UTC\n",
			dt.UTCDateTime.Date.Year,
			dt.UTCDateTime.Date.Month,
			dt.UTCDateTime.Date.Day,
			dt.UTCDateTime.Time.Hour,
			dt.UTCDateTime.Time.Minute,
			dt.UTCDateTime.Time.Second,
		)
	} else {
		fmt.Println("=== 设备信息 ===")
		fmt.Printf("制造商:       %s\n", info.Manufacturer)
		fmt.Printf("型号:         %s\n", info.Model)
		fmt.Printf("固件版本:     %s\n", info.FirmwareVersion)
		fmt.Printf("序列号:       %s\n", info.SerialNumber)
		fmt.Printf("硬件 ID:      %s\n", info.HardwareId)
	}

	return nil
}

// PTZ 移动
func (c *ONVIFClient) PTZMove(pan, tilt, zoom float64, timeout int) error {
	// 获取 profiles
	respData, err := c.sendRequest(c.MediaAddr, &GetProfiles{})
	if err != nil {
		return err
	}

	var profilesResp struct {
		Body struct {
			GetProfilesResponse GetProfilesResponse
		}
	}

	if err := xml.Unmarshal(respData, &profilesResp); err != nil {
		return fmt.Errorf("解析 profiles 失败: %w", err)
	}

	profiles := profilesResp.Body.GetProfilesResponse.Profiles
	if len(profiles) == 0 {
		return fmt.Errorf("设备没有可用的 profile")
	}

	// 构建 PTZ 服务地址
	ptzAddr := fmt.Sprintf("%s://%s:%d/onvif/ptz_service",
		map[bool]string{true: "https", false: "http"}[c.UseHTTPS], c.Host, c.Port)

	// 发送移动命令
	moveReq := ContinuousMove{
		ProfileToken: profiles[0].Token,
		Velocity: PTZSpeed{
			PanTilt: PanTilt{X: pan, Y: tilt},
			Zoom:    Zoom{X: zoom},
		},
	}

	if timeout > 0 {
		moveReq.Timeout = fmt.Sprintf("PT%dS", timeout)
	}

	_, err = c.sendRequest(ptzAddr, &moveReq)
	if err != nil {
		return fmt.Errorf("PTZ 移动失败: %w", err)
	}

	fmt.Printf("✓ PTZ 移动命令已发送\n")
	fmt.Printf("  水平速度: %.2f\n", pan)
	fmt.Printf("  垂直速度: %.2f\n", tilt)
	fmt.Printf("  缩放速度: %.2f\n", zoom)
	if timeout > 0 {
		fmt.Printf("  持续时间: %d 秒\n", timeout)
	}

	return nil
}

// PTZ 停止
func (c *ONVIFClient) PTZStop() error {
	// 获取 profiles
	respData, err := c.sendRequest(c.MediaAddr, &GetProfiles{})
	if err != nil {
		return err
	}

	var profilesResp struct {
		Body struct {
			GetProfilesResponse GetProfilesResponse
		}
	}

	if err := xml.Unmarshal(respData, &profilesResp); err != nil {
		return fmt.Errorf("解析 profiles 失败: %w", err)
	}

	profiles := profilesResp.Body.GetProfilesResponse.Profiles
	if len(profiles) == 0 {
		return fmt.Errorf("设备没有可用的 profile")
	}

	// 构建 PTZ 服务地址
	ptzAddr := fmt.Sprintf("%s://%s:%d/onvif/ptz_service",
		map[bool]string{true: "https", false: "http"}[c.UseHTTPS], c.Host, c.Port)

	// 发送停止命令
	stopReq := Stop{
		ProfileToken: profiles[0].Token,
		PanTilt:      true,
		Zoom:         true,
	}

	_, err = c.sendRequest(ptzAddr, &stopReq)
	if err != nil {
		return fmt.Errorf("PTZ 停止失败: %w", err)
	}

	fmt.Println("✓ PTZ 已停止")

	return nil
}

// PTZ 转到预置位
func (c *ONVIFClient) PTZGotoPreset(presetNum int) error {
	// 获取 profiles
	respData, err := c.sendRequest(c.MediaAddr, &GetProfiles{})
	if err != nil {
		return err
	}

	var profilesResp struct {
		Body struct {
			GetProfilesResponse GetProfilesResponse
		}
	}

	if err := xml.Unmarshal(respData, &profilesResp); err != nil {
		return fmt.Errorf("解析 profiles 失败: %w", err)
	}

	profiles := profilesResp.Body.GetProfilesResponse.Profiles
	if len(profiles) == 0 {
		return fmt.Errorf("设备没有可用的 profile")
	}

	// 构建 PTZ 服务地址
	ptzAddr := fmt.Sprintf("%s://%s:%d/onvif/ptz_service",
		map[bool]string{true: "https", false: "http"}[c.UseHTTPS], c.Host, c.Port)

	// 发送转到预置位命令
	gotoReq := GotoPreset{
		ProfileToken: profiles[0].Token,
		PresetToken:  fmt.Sprintf("%d", presetNum),
	}

	_, err = c.sendRequest(ptzAddr, &gotoReq)
	if err != nil {
		return fmt.Errorf("转到预置位失败: %w", err)
	}

	fmt.Printf("✓ 正在转到预置位 %d\n", presetNum)

	return nil
}

// PTZ 设置预置位
func (c *ONVIFClient) PTZSetPreset(presetNum int) error {
	// 获取 profiles
	respData, err := c.sendRequest(c.MediaAddr, &GetProfiles{})
	if err != nil {
		return err
	}

	var profilesResp struct {
		Body struct {
			GetProfilesResponse GetProfilesResponse
		}
	}

	if err := xml.Unmarshal(respData, &profilesResp); err != nil {
		return fmt.Errorf("解析 profiles 失败: %w", err)
	}

	profiles := profilesResp.Body.GetProfilesResponse.Profiles
	if len(profiles) == 0 {
		return fmt.Errorf("设备没有可用的 profile")
	}

	// 构建 PTZ 服务地址
	ptzAddr := fmt.Sprintf("%s://%s:%d/onvif/ptz_service",
		map[bool]string{true: "https", false: "http"}[c.UseHTTPS], c.Host, c.Port)

	// 发送设置预置位命令
	setReq := SetPreset{
		ProfileToken: profiles[0].Token,
		PresetName:   fmt.Sprintf("Preset_%d", presetNum),
	}

	respData, err = c.sendRequest(ptzAddr, &setReq)
	if err != nil {
		return fmt.Errorf("设置预置位失败: %w", err)
	}

	var setResp struct {
		Body struct {
			SetPresetResponse SetPresetResponse
		}
	}

	if err := xml.Unmarshal(respData, &setResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	fmt.Printf("✓ 预置位 %d 已设置\n", presetNum)
	fmt.Printf("  Token: %s\n", setResp.Body.SetPresetResponse.PresetToken)

	return nil
}

// PTZ 列出所有预置位
func (c *ONVIFClient) PTZListPresets() error {
	// 获取 profiles
	respData, err := c.sendRequest(c.MediaAddr, &GetProfiles{})
	if err != nil {
		return err
	}

	var profilesResp struct {
		Body struct {
			GetProfilesResponse GetProfilesResponse
		}
	}

	if err := xml.Unmarshal(respData, &profilesResp); err != nil {
		return fmt.Errorf("解析 profiles 失败: %w", err)
	}

	profiles := profilesResp.Body.GetProfilesResponse.Profiles
	if len(profiles) == 0 {
		return fmt.Errorf("设备没有可用的 profile")
	}

	// 构建 PTZ 服务地址
	ptzAddr := fmt.Sprintf("%s://%s:%d/onvif/ptz_service",
		map[bool]string{true: "https", false: "http"}[c.UseHTTPS], c.Host, c.Port)

	// 获取预置位列表
	getReq := GetPresets{
		ProfileToken: profiles[0].Token,
	}

	respData, err = c.sendRequest(ptzAddr, &getReq)
	if err != nil {
		return fmt.Errorf("获取预置位列表失败: %w", err)
	}

	var getResp struct {
		Body struct {
			GetPresetsResponse GetPresetsResponse
		}
	}

	if err := xml.Unmarshal(respData, &getResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	presets := getResp.Body.GetPresetsResponse.Presets

	fmt.Println("=== PTZ 预置位列表 ===")
	if len(presets) == 0 {
		fmt.Println("  (无预置位)")
	} else {
		for _, preset := range presets {
			fmt.Printf("  [%s] %s\n", preset.Token, preset.Name)
		}
	}

	return nil
}

// 获取抓图 URI 并下载
func (c *ONVIFClient) GetSnapshot(output string, profileIndex int) error {
	// 获取 profiles
	respData, err := c.sendRequest(c.MediaAddr, &GetProfiles{})
	if err != nil {
		return err
	}

	var profilesResp struct {
		Body struct {
			GetProfilesResponse GetProfilesResponse
		}
	}

	if err := xml.Unmarshal(respData, &profilesResp); err != nil {
		return fmt.Errorf("解析 profiles 失败: %w", err)
	}

	profiles := profilesResp.Body.GetProfilesResponse.Profiles
	if len(profiles) == 0 {
		return fmt.Errorf("设备没有可用的 profile")
	}

	if profileIndex >= len(profiles) {
		return fmt.Errorf("profile 索引 %d 超出范围 (0-%d)", profileIndex, len(profiles)-1)
	}

	// 获取抓图 URI
	snapshotReq := GetSnapshotUri{
		ProfileToken: profiles[profileIndex].Token,
	}

	uriData, err := c.sendRequest(c.MediaAddr, &snapshotReq)
	if err != nil {
		return fmt.Errorf("获取抓图 URI 失败: %w", err)
	}

	var uriResp struct {
		Body struct {
			GetSnapshotUriResponse GetSnapshotUriResponse
		}
	}

	if err := xml.Unmarshal(uriData, &uriResp); err != nil {
		return fmt.Errorf("解析抓图 URI 失败: %w", err)
	}

	snapshotURL := uriResp.Body.GetSnapshotUriResponse.MediaUri.Uri

	// 下载图像
	req, err := http.NewRequest("GET", snapshotURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加认证
	if c.AuthMode == "digest" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("下载图像失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载图像失败，状态码: %d", resp.StatusCode)
	}

	// 保存到文件
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取图像数据失败: %w", err)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("保存图像失败: %w", err)
	}

	fmt.Printf("✓ 图像已保存到: %s\n", output)
	fmt.Printf("  配置: %s\n", profiles[profileIndex].Name)
	fmt.Printf("  大小: %d 字节\n", len(data))

	return nil
}

// 获取视频编码配置
func (c *ONVIFClient) GetVideoEncoderConfiguration() error {
	respData, err := c.sendRequest(c.MediaAddr, &GetVideoEncoderConfigurations{})
	if err != nil {
		return err
	}

	var configResp struct {
		Body struct {
			GetVideoEncoderConfigurationsResponse GetVideoEncoderConfigurationsResponse
		}
	}

	if err = xml.Unmarshal(respData, &configResp); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	configs := configResp.Body.GetVideoEncoderConfigurationsResponse.Configurations

	fmt.Println("=== 视频编码配置 ===")
	for i, config := range configs {
		fmt.Printf("\n配置 %d:\n", i)
		fmt.Printf("  Token:      %s\n", config.Token)
		fmt.Printf("  名称:       %s\n", config.Name)
		fmt.Printf("  编码:       %s\n", config.Encoding)
		fmt.Printf("  分辨率:     %dx%d\n", config.Resolution.Width, config.Resolution.Height)
		fmt.Printf("  质量:       %v\n", config.Quality)
		fmt.Printf("  帧率:       %d fps\n", config.RateControl.FrameRateLimit)
		fmt.Printf("  比特率:     %d kbps\n", config.RateControl.BitrateLimit)
	}

	return nil
}

// 设置视频编码配置
func (c *ONVIFClient) SetVideoEncoderConfiguration(width, height, fps, bitrate int) error {
	// 先获取当前配置
	respData, err := c.sendRequest(c.MediaAddr, &GetVideoEncoderConfigurations{})
	if err != nil {
		return err
	}

	var configResp struct {
		Body struct {
			GetVideoEncoderConfigurationsResponse GetVideoEncoderConfigurationsResponse
		}
	}

	if err := xml.Unmarshal(respData, &configResp); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	configs := configResp.Body.GetVideoEncoderConfigurationsResponse.Configurations
	if len(configs) == 0 {
		return fmt.Errorf("没有找到视频编码配置")
	}

	// 修改第一个配置
	config := configs[0]

	if width > 0 {
		config.Resolution.Width = width
	}
	if height > 0 {
		config.Resolution.Height = height
	}
	if fps > 0 {
		config.RateControl.FrameRateLimit = fps
	}
	if bitrate > 0 {
		config.RateControl.BitrateLimit = bitrate
	}

	// 发送设置请求
	setReq := SetVideoEncoderConfiguration{
		Configuration:    config,
		ForcePersistence: true,
	}

	_, err = c.sendRequest(c.MediaAddr, &setReq)
	if err != nil {
		return fmt.Errorf("设置配置失败: %w", err)
	}

	fmt.Println("✓ 视频编码配置已更新")
	fmt.Printf("  分辨率: %dx%d\n", config.Resolution.Width, config.Resolution.Height)
	fmt.Printf("  帧率:   %d fps\n", config.RateControl.FrameRateLimit)
	fmt.Printf("  比特率: %d kbps\n", config.RateControl.BitrateLimit)

	return nil
}

// 获取网络配置
func (c *ONVIFClient) GetNetworkConfiguration() error {
	respData, err := c.sendRequest(c.XAddr, &GetNetworkInterfaces{})
	if err != nil {
		return err
	}

	var netResp struct {
		Body struct {
			GetNetworkInterfacesResponse GetNetworkInterfacesResponse
		}
	}

	if err := xml.Unmarshal(respData, &netResp); err != nil {
		return fmt.Errorf("解析网络配置失败: %w", err)
	}

	interfaces := netResp.Body.GetNetworkInterfacesResponse.NetworkInterfaces

	fmt.Println("=== 网络配置 ===")
	for _, iface := range interfaces {
		fmt.Printf("\n接口: %s\n", iface.Info.Name)
		fmt.Printf("  Token:      %s\n", iface.Token)
		fmt.Printf("  启用:       %t\n", iface.Enabled)
		fmt.Printf("  MAC 地址:   %s\n", iface.Info.HwAddress)
		fmt.Printf("  MTU:        %d\n", iface.Info.MTU)
		fmt.Printf("  IPv4 启用:  %t\n", iface.IPv4.Enabled)
		fmt.Printf("  DHCP:       %t\n", iface.IPv4.DHCP)

		if len(iface.IPv4.Manual) > 0 {
			fmt.Println("  手动 IP:")
			for _, addr := range iface.IPv4.Manual {
				fmt.Printf("    %s/%d\n", addr.Address, addr.PrefixLength)
			}
		}
	}

	return nil
}

// 获取流地址
func (c *ONVIFClient) GetStreamURI(profileIndex int) error {
	// 获取所有 profiles
	respData, err := c.sendRequest(c.MediaAddr, &GetProfiles{})
	if err != nil {
		return err
	}

	var profilesResp struct {
		Body struct {
			GetProfilesResponse GetProfilesResponse
		}
	}

	if err := xml.Unmarshal(respData, &profilesResp); err != nil {
		return fmt.Errorf("解析 profiles 失败: %w", err)
	}

	profiles := profilesResp.Body.GetProfilesResponse.Profiles
	if len(profiles) == 0 {
		return fmt.Errorf("设备没有可用的 profile")
	}

	if profileIndex >= len(profiles) {
		return fmt.Errorf("profile 索引 %d 超出范围 (0-%d)", profileIndex, len(profiles)-1)
	}

	// 获取流 URI
	streamReq := GetStreamUri{
		StreamSetup: StreamSetup{
			Stream: "RTP-Unicast",
			Transport: Transport{
				Protocol: "RTSP",
			},
		},
		ProfileToken: profiles[profileIndex].Token,
	}

	uriData, err := c.sendRequest(c.MediaAddr, &streamReq)
	if err != nil {
		return err
	}

	var uriResp struct {
		Body struct {
			GetStreamUriResponse GetStreamUriResponse
		}
	}

	if err := xml.Unmarshal(uriData, &uriResp); err != nil {
		return fmt.Errorf("解析流地址失败: %w", err)
	}

	fmt.Println("=== 视频流信息 ===")
	fmt.Printf("配置名称:     %s\n", profiles[profileIndex].Name)
	fmt.Printf("配置 Token:   %s\n", profiles[profileIndex].Token)
	fmt.Printf("RTSP 地址:    %s\n", uriResp.Body.GetStreamUriResponse.MediaUri.Uri)

	fmt.Println("\n所有可用配置:")
	for i, p := range profiles {
		fmt.Printf("  [%d] %s (Token: %s)\n", i, p.Name, p.Token)
	}

	return nil
}
