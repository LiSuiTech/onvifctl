package main

import (
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// WS-Discovery 相关结构
type ProbeMatch struct {
	Address string
	Types   string
	XAddrs  string
	Scopes  string
}

// 设备发现
func DiscoverDevices(timeout int, ifaceName string, debug bool) ([]ProbeMatch, error) {
	// 构建 WS-Discovery Probe 消息
	probeMsg := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" 
            xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery">
  <s:Header>
    <a:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</a:Action>
    <a:MessageID>uuid:` + generateUUID() + `</a:MessageID>
    <a:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</a:To>
  </s:Header>
  <s:Body>
    <d:Probe>
      <d:Types>dn:NetworkVideoTransmitter</d:Types>
    </d:Probe>
  </s:Body>
</s:Envelope>`

	// 创建 UDP 连接
	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:3702")
	if err != nil {
		return nil, fmt.Errorf("解析地址失败: %w", err)
	}

	var conn *net.UDPConn
	if ifaceName != "" {
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			return nil, fmt.Errorf("找不到网络接口 %s: %w", ifaceName, err)
		}
		conn, err = net.ListenMulticastUDP("udp4", iface, addr)
	} else {
		conn, err = net.ListenMulticastUDP("udp4", nil, addr)
	}

	if err != nil {
		return nil, fmt.Errorf("创建 UDP 连接失败: %w", err)
	}
	defer conn.Close()

	// 发送 Probe 消息
	if debug {
		fmt.Println("发送设备发现请求...")
	}

	_, err = conn.WriteToUDP([]byte(probeMsg), addr)
	if err != nil {
		return nil, fmt.Errorf("发送探测消息失败: %w", err)
	}

	// 接收响应
	conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	devices := make(map[string]ProbeMatch)
	buffer := make([]byte, 8192)

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			continue
		}

		// 解析响应
		response := string(buffer[:n])
		if debug {
			fmt.Printf("\n收到响应:\n%s\n", response)
		}

		// 简单解析 XML 提取关键信息
		match := ProbeMatch{}

		// 提取 Address
		if start := strings.Index(response, "<a:Address>"); start != -1 {
			start += 11
			if end := strings.Index(response[start:], "</a:Address>"); end != -1 {
				match.Address = response[start : start+end]
			}
		}

		// 提取 XAddrs
		if start := strings.Index(response, "<d:XAddrs>"); start != -1 {
			start += 10
			if end := strings.Index(response[start:], "</d:XAddrs>"); end != -1 {
				match.XAddrs = response[start : start+end]
			}
		}

		// 提取 Types
		if start := strings.Index(response, "<d:Types>"); start != -1 {
			start += 9
			if end := strings.Index(response[start:], "</d:Types>"); end != -1 {
				match.Types = response[start : start+end]
			}
		}

		// 提取 Scopes
		if start := strings.Index(response, "<d:Scopes>"); start != -1 {
			start += 10
			if end := strings.Index(response[start:], "</d:Scopes>"); end != -1 {
				match.Scopes = response[start : start+end]
			}
		}

		if match.XAddrs != "" {
			devices[match.Address] = match
		}
	}

	// 转换为切片
	result := make([]ProbeMatch, 0, len(devices))
	for _, device := range devices {
		result = append(result, device)
	}

	return result, nil
}

// 生成 UUID
func generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano()&0xFFFFFFFF,
		time.Now().UnixNano()>>32&0xFFFF,
		0x4000|(time.Now().UnixNano()>>48&0x0FFF),
		0x8000|(time.Now().UnixNano()>>60&0x3FFF),
		time.Now().UnixNano()&0xFFFFFFFFFFFF)
}

// 保存设备列表到文件
func SaveDevicesToFile(devices []ProbeMatch, filename string) error {
	data, err := yaml.Marshal(devices)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// 时间同步相关结构
type SetSystemDateAndTime struct {
	XMLName         xml.Name     `xml:"http://www.onvif.org/ver10/device/wsdl SetSystemDateAndTime"`
	DateTimeType    string       `xml:"DateTimeType"`
	DaylightSavings bool         `xml:"DaylightSavings"`
	UTCDateTime     *UTCDateTime `xml:"UTCDateTime,omitempty"`
}

type SetNTP struct {
	XMLName   xml.Name      `xml:"http://www.onvif.org/ver10/device/wsdl SetNTP"`
	FromDHCP  bool          `xml:"FromDHCP"`
	NTPManual []NetworkHost `xml:"NTPManual"`
}

type NetworkHost struct {
	Type        string `xml:"Type"`
	IPv4Address string `xml:"IPv4Address,omitempty"`
	DNSname     string `xml:"DNSname,omitempty"`
}

// 获取系统时间
func (c *ONVIFClient) GetSystemTime() error {
	respData, err := c.sendRequest(c.XAddr, &GetSystemDateAndTime{})
	if err != nil {
		return err
	}

	var timeResp struct {
		Body struct {
			GetSystemDateAndTimeResponse GetSystemDateAndTimeResponse
		}
	}

	if err := xml.Unmarshal(respData, &timeResp); err != nil {
		return fmt.Errorf("解析时间失败: %w", err)
	}

	dt := timeResp.Body.GetSystemDateAndTimeResponse.SystemDateAndTime

	fmt.Println("=== 设备时间 ===")
	fmt.Printf("时间类型: %s\n", dt.DateTimeType)
	fmt.Printf("设备时间: %04d-%02d-%02d %02d:%02d:%02d UTC\n",
		dt.UTCDateTime.Date.Year,
		dt.UTCDateTime.Date.Month,
		dt.UTCDateTime.Date.Day,
		dt.UTCDateTime.Time.Hour,
		dt.UTCDateTime.Time.Minute,
		dt.UTCDateTime.Time.Second,
	)

	// 显示与系统时间的差异
	deviceTime := time.Date(
		dt.UTCDateTime.Date.Year,
		time.Month(dt.UTCDateTime.Date.Month),
		dt.UTCDateTime.Date.Day,
		dt.UTCDateTime.Time.Hour,
		dt.UTCDateTime.Time.Minute,
		dt.UTCDateTime.Time.Second,
		0, time.UTC,
	)

	diff := time.Since(deviceTime)
	fmt.Printf("时间差异: %.0f 秒\n", diff.Seconds())

	return nil
}

// 同步系统时间
func (c *ONVIFClient) SyncSystemTime() error {
	now := time.Now().UTC()

	setTimeReq := SetSystemDateAndTime{
		DateTimeType:    "Manual",
		DaylightSavings: false,
		UTCDateTime: &UTCDateTime{
			Time: Time{
				Hour:   now.Hour(),
				Minute: now.Minute(),
				Second: now.Second(),
			},
			Date: Date{
				Year:  now.Year(),
				Month: int(now.Month()),
				Day:   now.Day(),
			},
		},
	}

	_, err := c.sendRequest(c.XAddr, &setTimeReq)
	if err != nil {
		return fmt.Errorf("同步时间失败: %w", err)
	}

	fmt.Println("✓ 时间同步成功")
	fmt.Printf("  设备时间已设置为: %s\n", now.Format("2006-01-02 15:04:05 UTC"))

	return nil
}

// 设置 NTP 服务器
func (c *ONVIFClient) SetNTP(ntpServer string) error {
	setNTPReq := SetNTP{
		FromDHCP: false,
		NTPManual: []NetworkHost{
			{
				Type:        "IPv4",
				IPv4Address: ntpServer,
			},
		},
	}

	_, err := c.sendRequest(c.XAddr, &setNTPReq)
	if err != nil {
		return fmt.Errorf("设置 NTP 失败: %w", err)
	}

	fmt.Println("✓ NTP 服务器设置成功")
	fmt.Printf("  NTP 服务器: %s\n", ntpServer)

	return nil
}

// 事件订阅
func (c *ONVIFClient) SubscribeEvents(duration int, filter string) error {
	fmt.Printf("订阅设备事件 (持续 %d 秒)...\n", duration)
	fmt.Println("注意: 完整的事件订阅需要 WS-BaseNotification 和 PullPoint 支持")
	fmt.Println("这是一个简化实现，实际生产环境建议使用专门的事件处理库")

	// 这里是一个简化的实现框架
	// 完整实现需要 CreatePullPointSubscription 和持续的 PullMessages

	eventAddr := fmt.Sprintf("%s://%s:%d/onvif/event_service",
		map[bool]string{true: "https", false: "http"}[c.UseHTTPS], c.Host, c.Port)

	fmt.Printf("事件服务地址: %s\n", eventAddr)
	fmt.Println("✓ 事件订阅功能需要设备支持 ONVIF Events Profile")

	return nil
}

// 批量配置管理
type BatchConfig struct {
	Devices []DeviceConfig `yaml:"devices"`
}

type DeviceConfig struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	UseHTTPS bool   `yaml:"use_https"`
}

// 加载批量配置
func LoadBatchConfig(filename string) (*BatchConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config BatchConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// 保存批量配置
func SaveBatchConfig(config *BatchConfig, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// 批量获取设备信息
func BatchGetInfo(config *BatchConfig) error {
	fmt.Printf("正在获取 %d 个设备的信息...\n\n", len(config.Devices))

	var wg sync.WaitGroup
	results := make(chan string, len(config.Devices))

	for i, device := range config.Devices {
		wg.Add(1)
		go func(idx int, dev DeviceConfig) {
			defer wg.Done()

			client, err := NewONVIFClient(dev.Host, dev.Port, dev.Username, dev.Password, false, dev.UseHTTPS)
			if err != nil {
				results <- fmt.Sprintf("[%d] %s - 连接失败: %v", idx+1, dev.Name, err)
				return
			}

			// 获取设备信息
			respData, err := client.sendRequest(client.XAddr, &GetDeviceInformation{})
			if err != nil {
				results <- fmt.Sprintf("[%d] %s - 获取信息失败: %v", idx+1, dev.Name, err)
				return
			}

			var infoResp struct {
				Body struct {
					GetDeviceInformationResponse GetDeviceInformationResponse
				}
			}

			if err := xml.Unmarshal(respData, &infoResp); err != nil {
				results <- fmt.Sprintf("[%d] %s - 解析失败: %v", idx+1, dev.Name, err)
				return
			}

			info := infoResp.Body.GetDeviceInformationResponse
			results <- fmt.Sprintf("[%d] %s - %s %s (固件: %s)",
				idx+1, dev.Name, info.Manufacturer, info.Model, info.FirmwareVersion)
		}(i, device)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 打印结果
	for result := range results {
		fmt.Println(result)
	}

	fmt.Println("\n✓ 批量查询完成")
	return nil
}

// 批量抓图
func BatchSnapshot(config *BatchConfig, outputDir string) error {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	fmt.Printf("正在从 %d 个设备抓取图像...\n\n", len(config.Devices))

	var wg sync.WaitGroup
	results := make(chan string, len(config.Devices))

	for i, device := range config.Devices {
		wg.Add(1)
		go func(idx int, dev DeviceConfig) {
			defer wg.Done()

			client, err := NewONVIFClient(dev.Host, dev.Port, dev.Username, dev.Password, false, dev.UseHTTPS)
			if err != nil {
				results <- fmt.Sprintf("[%d] %s - 连接失败: %v", idx+1, dev.Name, err)
				return
			}

			outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.jpg", dev.Name))
			if err := client.GetSnapshot(outputFile, 0); err != nil {
				results <- fmt.Sprintf("[%d] %s - 抓图失败: %v", idx+1, dev.Name, err)
				return
			}

			results <- fmt.Sprintf("[%d] %s - ✓ 已保存到 %s", idx+1, dev.Name, outputFile)
		}(i, device)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 打印结果
	for result := range results {
		fmt.Println(result)
	}

	fmt.Printf("\n✓ 批量抓图完成，图像保存在: %s\n", outputDir)
	return nil
}

// 批量同步时间
func BatchSyncTime(config *BatchConfig) error {
	fmt.Printf("正在同步 %d 个设备的时间...\n\n", len(config.Devices))

	var wg sync.WaitGroup
	results := make(chan string, len(config.Devices))

	for i, device := range config.Devices {
		wg.Add(1)
		go func(idx int, dev DeviceConfig) {
			defer wg.Done()

			client, err := NewONVIFClient(dev.Host, dev.Port, dev.Username, dev.Password, false, dev.UseHTTPS)
			if err != nil {
				results <- fmt.Sprintf("[%d] %s - 连接失败: %v", idx+1, dev.Name, err)
				return
			}

			if err := client.SyncSystemTime(); err != nil {
				results <- fmt.Sprintf("[%d] %s - 同步失败: %v", idx+1, dev.Name, err)
				return
			}

			results <- fmt.Sprintf("[%d] %s - ✓ 时间同步成功", idx+1, dev.Name)
		}(i, device)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 打印结果
	for result := range results {
		fmt.Println(result)
	}

	fmt.Println("\n✓ 批量时间同步完成")
	return nil
}
