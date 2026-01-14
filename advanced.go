package main

import (
	"crypto/rand"
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

const (
	// WS-Discovery 组播地址和端口
	wsDiscoveryMulticastAddr = "239.255.255.250:3702"
	wsDiscoveryTimeout       = 3 * time.Second

	// 文件权限
	configFilePermission = 0644
	snapshotDirPermission = 0755
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

// 生成 UUID v4 (标准实现)
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// 降级到基于时间的 UUID
		nano := time.Now().UnixNano()
		for i := 0; i < 16; i++ {
			uuid[i] = byte(nano >> (i * 8))
		}
	}

	// 设置版本 (4) 和变体 (RFC 4122)
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // 版本 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // 变体 RFC 4122

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(uuid[0])<<24|uint32(uuid[1])<<16|uint32(uuid[2])<<8|uint32(uuid[3]),
		uint16(uuid[4])<<8|uint16(uuid[5]),
		uint16(uuid[6])<<8|uint16(uuid[7]),
		uint16(uuid[8])<<8|uint16(uuid[9]),
		uint64(uuid[10])<<40|uint64(uuid[11])<<32|uint64(uuid[12])<<24|uint64(uuid[13])<<16|uint64(uuid[14])<<8|uint64(uuid[15]))
}

// 保存设备列表到文件
func SaveDevicesToFile(devices []ProbeMatch, filename string) error {
	data, err := yaml.Marshal(devices)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, configFilePermission)
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

// 事件订阅相关结构
type CreatePullPointSubscription struct {
	XMLName                xml.Name `xml:"http://www.onvif.org/ver10/events/wsdl CreatePullPointSubscription"`
	Filter                 *Filter  `xml:"Filter,omitempty"`
	InitialTerminationTime string   `xml:"InitialTerminationTime,omitempty"`
}

type Filter struct {
	TopicExpression string `xml:"TopicExpression,omitempty"`
}

type CreatePullPointSubscriptionResponse struct {
	SubscriptionReference SubscriptionReference `xml:"SubscriptionReference"`
	CurrentTime           string                `xml:"CurrentTime"`
	TerminationTime       string                `xml:"TerminationTime"`
}

type SubscriptionReference struct {
	Address string `xml:"Address"`
}

type PullMessages struct {
	XMLName      xml.Name `xml:"http://www.onvif.org/ver10/events/wsdl PullMessages"`
	Timeout      string   `xml:"Timeout"`
	MessageLimit int      `xml:"MessageLimit"`
}

type PullMessagesResponse struct {
	CurrentTime         string                `xml:"CurrentTime"`
	TerminationTime     string                `xml:"TerminationTime"`
	NotificationMessage []NotificationMessage `xml:"NotificationMessage"`
}

type NotificationMessage struct {
	Topic   Topic   `xml:"Topic"`
	Message Message `xml:"Message"`
}

type Topic struct {
	Dialect string `xml:"Dialect,attr"`
	Value   string `xml:",chardata"`
}

type Message struct {
	UtcTime string      `xml:"UtcTime,attr"`
	Source  EventSource `xml:"Source"`
	Data    EventData   `xml:"Data"`
}

type EventSource struct {
	SimpleItem []SimpleItem `xml:"SimpleItem"`
}

type EventData struct {
	SimpleItem []SimpleItem `xml:"SimpleItem"`
}

type SimpleItem struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"Value,attr"`
}

type Renew struct {
	XMLName         xml.Name `xml:"http://docs.oasis-open.org/wsn/b-2 Renew"`
	TerminationTime string   `xml:"TerminationTime"`
}

type RenewResponse struct {
	TerminationTime string `xml:"TerminationTime"`
	CurrentTime     string `xml:"CurrentTime"`
}

type Unsubscribe struct {
	XMLName xml.Name `xml:"http://docs.oasis-open.org/wsn/b-2 Unsubscribe"`
}

// 事件订阅完整实现
func (c *ONVIFClient) SubscribeEvents(duration int, filter string) error {
	fmt.Printf("正在订阅设备事件 (持续 %d 秒)...\n\n", duration)

	eventAddr := fmt.Sprintf("%s://%s:%d/onvif/event_service",
		map[bool]string{true: "https", false: "http"}[c.UseHTTPS], c.Host, c.Port)

	// 1. 创建 PullPoint 订阅
	fmt.Println("步骤 1: 创建 PullPoint 订阅...")

	subscribeReq := CreatePullPointSubscription{
		InitialTerminationTime: fmt.Sprintf("PT%dS", duration),
	}

	// 如果指定了过滤器
	if filter != "" {
		subscribeReq.Filter = &Filter{
			TopicExpression: filter,
		}
	}

	respData, err := c.sendRequest(eventAddr, &subscribeReq)
	if err != nil {
		return fmt.Errorf("创建订阅失败: %w", err)
	}

	var subscribeResp struct {
		Body struct {
			CreatePullPointSubscriptionResponse CreatePullPointSubscriptionResponse
		}
	}

	if err = xml.Unmarshal(respData, &subscribeResp); err != nil {
		return fmt.Errorf("解析订阅响应失败: %w", err)
	}

	subRef := subscribeResp.Body.CreatePullPointSubscriptionResponse
	pullPointAddr := subRef.SubscriptionReference.Address

	fmt.Printf("✓ 订阅成功\n")
	fmt.Printf("  订阅地址: %s\n", pullPointAddr)
	fmt.Printf("  当前时间: %s\n", subRef.CurrentTime)
	fmt.Printf("  终止时间: %s\n\n", subRef.TerminationTime)

	// 2. 开始拉取消息
	fmt.Println("步骤 2: 开始监听事件...")
	fmt.Println("----------------------------------------")

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(duration) * time.Second)
	messageCount := 0
	renewInterval := time.Duration(duration/2) * time.Second
	lastRenew := startTime

	for time.Now().Before(endTime) {
		// 定期续订
		if time.Since(lastRenew) > renewInterval {
			if err = c.renewSubscription(pullPointAddr, duration); err != nil {
				fmt.Printf("⚠ 续订失败: %v\n", err)
			} else {
				lastRenew = time.Now()
				if c.Debug {
					fmt.Println("✓ 订阅已续订")
				}
			}
		}

		// 拉取消息
		pullReq := PullMessages{
			Timeout:      "PT5S",
			MessageLimit: 10,
		}

		respData, err = c.sendRequest(pullPointAddr, &pullReq)
		if err != nil {
			if c.Debug {
				fmt.Printf("拉取消息失败: %v\n", err)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		var pullResp struct {
			Body struct {
				PullMessagesResponse PullMessagesResponse
			}
		}

		if err = xml.Unmarshal(respData, &pullResp); err != nil {
			if c.Debug {
				fmt.Printf("解析消息失败: %v\n", err)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		messages := pullResp.Body.PullMessagesResponse.NotificationMessage

		// 处理接收到的消息
		for _, msg := range messages {
			messageCount++
			c.printEventMessage(messageCount, msg)
		}

		// 如果没有消息，短暂休眠
		if len(messages) == 0 {
			time.Sleep(1 * time.Second)
		}
	}

	// 3. 取消订阅
	fmt.Println("\n----------------------------------------")
	fmt.Println("步骤 3: 取消订阅...")

	if err := c.unsubscribe(pullPointAddr); err != nil {
		fmt.Printf("⚠ 取消订阅失败: %v\n", err)
	} else {
		fmt.Println("✓ 订阅已取消")
	}

	fmt.Printf("\n事件监听完成，共接收 %d 条消息\n", messageCount)

	return nil
}

// 续订订阅
func (c *ONVIFClient) renewSubscription(pullPointAddr string, duration int) error {
	renewReq := Renew{
		TerminationTime: fmt.Sprintf("PT%dS", duration),
	}

	respData, err := c.sendRequest(pullPointAddr, &renewReq)
	if err != nil {
		return err
	}

	var renewResp struct {
		Body struct {
			RenewResponse RenewResponse
		}
	}

	if err := xml.Unmarshal(respData, &renewResp); err != nil {
		return err
	}

	return nil
}

// 取消订阅
func (c *ONVIFClient) unsubscribe(pullPointAddr string) error {
	unsubReq := Unsubscribe{}
	_, err := c.sendRequest(pullPointAddr, &unsubReq)
	return err
}

// 打印事件消息
func (c *ONVIFClient) printEventMessage(count int, msg NotificationMessage) {
	fmt.Printf("\n[事件 #%d]\n", count)
	fmt.Printf("时间: %s\n", msg.Message.UtcTime)
	fmt.Printf("主题: %s\n", msg.Topic.Value)

	// 打印事件源
	if len(msg.Message.Source.SimpleItem) > 0 {
		fmt.Println("来源:")
		for _, item := range msg.Message.Source.SimpleItem {
			fmt.Printf("  %s: %s\n", item.Name, item.Value)
		}
	}

	// 打印事件数据
	if len(msg.Message.Data.SimpleItem) > 0 {
		fmt.Println("数据:")
		for _, item := range msg.Message.Data.SimpleItem {
			fmt.Printf("  %s: %s\n", item.Name, item.Value)
		}
	}

	// 解析常见事件类型
	c.parseEventType(msg)
}

// 解析事件类型
func (c *ONVIFClient) parseEventType(msg NotificationMessage) {
	topic := msg.Topic.Value

	// 运动检测
	if strings.Contains(topic, "MotionDetector") || strings.Contains(topic, "CellMotionDetector") {
		for _, item := range msg.Message.Data.SimpleItem {
			if item.Name == "State" {
				if item.Value == "true" {
					fmt.Println(">>> 检测到运动!")
				} else {
					fmt.Println(">>> 运动结束")
				}
			}
		}
	}

	// 篡改检测
	if strings.Contains(topic, "TamperDetector") {
		for _, item := range msg.Message.Data.SimpleItem {
			if item.Name == "State" {
				if item.Value == "true" {
					fmt.Println(">>> 检测到篡改!")
				} else {
					fmt.Println(">>> 篡改结束")
				}
			}
		}
	}

	// 音频检测
	if strings.Contains(topic, "AudioAnalytics") {
		fmt.Println(">>> 音频事件触发")
	}

	// 区域入侵
	if strings.Contains(topic, "FieldDetector") {
		fmt.Println(">>> 区域入侵检测")
	}

	// 越界检测
	if strings.Contains(topic, "LineDetector") {
		fmt.Println(">>> 越界检测")
	}
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

	return os.WriteFile(filename, data, configFilePermission)
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
	if err := os.MkdirAll(outputDir, snapshotDirPermission); err != nil {
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
