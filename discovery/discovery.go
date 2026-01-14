package discovery

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

const (
	bufSize = 8192

	// 默认端口列表
	defaultPorts = "80,8080,8000,8899,9000,554"

	// 默认超时时间
	defaultProbeTimeout = 2 * time.Second

	// 并发控制
	maxConcurrentProbes = 1000

	// 进度显示间隔
	progressUpdateInterval = 500

	// WS-Discovery 相关
	wsDiscoveryMulticastIP   = "239.255.255.250"
	wsDiscoveryPort          = 3702
	wsDiscoveryTimeout       = 3 * time.Second
	wsDiscoveryMulticastTTL  = 2
)

// ONVIFDevice ONVIF设备基本信息
type ONVIFDevice struct {
	XAddr string // 设备地址
	IP    string // IP地址
	Port  int    // 端口
	Path  string // 路径
}

// DeviceDiscovery 设备发现器
type DeviceDiscovery struct {
	devices []ONVIFDevice
}

// IPRange IP范围
type IPRange struct {
	StartIP string
	EndIP   string
}

// NewDeviceDiscovery 创建设备发现器
func NewDeviceDiscovery() *DeviceDiscovery {
	return &DeviceDiscovery{
		devices: make([]ONVIFDevice, 0),
	}
}

// DiscoverByBroadcast 通过广播发现设备
func (dd *DeviceDiscovery) DiscoverByBroadcast(interfaceName string) ([]ONVIFDevice, error) {
	fmt.Println("开始广播发现ONVIF设备...")

	msg := dd.buildProbeMessage()
	responses := dd.sendUDPMulticast(msg, interfaceName)

	// 使用 IP:Port 作为唯一标识去重
	deviceMap := make(map[string]ONVIFDevice)
	dd.devices = make([]ONVIFDevice, 0)

	for _, response := range responses {
		xaddrs := dd.parseProbeResponse(response)
		for _, xaddr := range xaddrs {
			device := dd.parseDeviceInfo(xaddr)
			key := fmt.Sprintf("%s:%d", device.IP, device.Port)

			// 如果该IP:端口还没有记录,或者当前路径更短(优先选择更简洁的路径)
			if existing, exists := deviceMap[key]; !exists || len(device.Path) < len(existing.Path) {
				deviceMap[key] = device
			}
		}
	}

	// 转换为切片
	for _, device := range deviceMap {
		dd.devices = append(dd.devices, device)
	}

	fmt.Printf("广播发现完成,找到 %d 个设备\n", len(dd.devices))
	return dd.devices, nil
}

// DiscoverByIPRange 通过IP范围扫描发现设备
func (dd *DeviceDiscovery) DiscoverByIPRange(startIP, endIP string, ports []int, timeout time.Duration) ([]ONVIFDevice, error) {
	fmt.Printf("开始扫描IP范围: %s - %s\n", startIP, endIP)

	if len(ports) == 0 {
		ports = []int{80, 8080, 8000, 8899, 9000, 554}
	}

	if timeout == 0 {
		timeout = defaultProbeTimeout
	}

	ips, err := dd.generateIPRange(startIP, endIP)
	if err != nil {
		return nil, err
	}

	fmt.Printf("需要扫描 %d 个IP地址, %d 个端口\n", len(ips), len(ports))

	// 使用 IP:Port 作为唯一标识去重
	deviceMap := make(map[string]ONVIFDevice)
	dd.devices = make([]ONVIFDevice, 0)

	// ONVIF 常见路径
	paths := []string{
		"/onvif/device_service",
		"/onvif/services",
		"/ONVIF/device_service",
		"/onvif-http/services",
	}

	results := make(chan ONVIFDevice, len(ips)*len(ports)*len(paths))
	semaphore := make(chan struct{}, maxConcurrentProbes)

	var wg sync.WaitGroup
	var mu sync.Mutex // 用于保护 deviceMap

	// 为了更快的反馈,添加进度显示
	totalTasks := len(ips) * len(ports) * len(paths)
	completedTasks := 0
	progressMu := sync.Mutex{}

	for _, ip := range ips {
		for _, port := range ports {
			for _, path := range paths {
				wg.Add(1)
				go func(ip string, port int, path string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() {
						<-semaphore

						// 更新进度
						progressMu.Lock()
						completedTasks++
						if completedTasks%progressUpdateInterval == 0 || completedTasks == totalTasks {
							fmt.Printf("\r扫描进度: %d/%d (%.1f%%)  ", completedTasks, totalTasks, float64(completedTasks)*100/float64(totalTasks))
						}
						progressMu.Unlock()
					}()

					xaddr := fmt.Sprintf("http://%s:%d%s", ip, port, path)
					if dd.probeDevice(xaddr, timeout) {
						device := ONVIFDevice{
							XAddr: xaddr,
							IP:    ip,
							Port:  port,
							Path:  path,
						}
						results <- device
					}
				}(ip, port, path)
			}
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果并去重
	for device := range results {
		key := fmt.Sprintf("%s:%d", device.IP, device.Port)

		mu.Lock()
		// 如果该IP:端口还没有记录,或者当前路径更短(优先选择更简洁的路径)
		if existing, exists := deviceMap[key]; !exists {
			deviceMap[key] = device
		} else if len(device.Path) < len(existing.Path) {
			// 如果新路径更短,替换
			deviceMap[key] = device
		}
		mu.Unlock()
	}

	// 转换为切片
	fmt.Println() // 换行,避免覆盖进度条
	for _, device := range deviceMap {
		dd.devices = append(dd.devices, device)
	}

	return dd.devices, nil
}

// DiscoverMixed 混合模式发现
func (dd *DeviceDiscovery) DiscoverMixed(interfaceName string, ipRanges []IPRange, ports []int, timeout time.Duration) ([]ONVIFDevice, error) {
	// 使用 IP:Port 作为唯一标识去重
	deviceMap := make(map[string]ONVIFDevice)

	// 1. 广播发现
	broadcastDevices, err := dd.DiscoverByBroadcast(interfaceName)
	if err == nil {
		for _, dev := range broadcastDevices {
			key := fmt.Sprintf("%s:%d", dev.IP, dev.Port)
			deviceMap[key] = dev
		}
	}

	// 2. IP范围扫描
	if len(ipRanges) > 0 {
		for _, ipRange := range ipRanges {
			rangeDevices, err := dd.DiscoverByIPRange(ipRange.StartIP, ipRange.EndIP, ports, timeout)
			if err != nil {
				continue
			}

			for _, dev := range rangeDevices {
				key := fmt.Sprintf("%s:%d", dev.IP, dev.Port)
				// 如果该IP:端口还没有记录,或者当前路径更短
				if existing, exists := deviceMap[key]; !exists || len(dev.Path) < len(existing.Path) {
					deviceMap[key] = dev
				}
			}
		}
	}

	// 转换为切片
	dd.devices = make([]ONVIFDevice, 0, len(deviceMap))
	for _, device := range deviceMap {
		dd.devices = append(dd.devices, device)
	}

	return dd.devices, nil
}

// GetDevices 获取已发现的设备列表
func (dd *DeviceDiscovery) GetDevices() []ONVIFDevice {
	return dd.devices
}

// 内部方法

func (dd *DeviceDiscovery) buildProbeMessage() string {
	uuid := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
	<s:Header>
		<a:Action s:mustUnderstand="1">http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</a:Action>
		<a:MessageID>uuid:%s</a:MessageID>
		<a:ReplyTo>
			<a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address>
		</a:ReplyTo>
		<a:To s:mustUnderstand="1">urn:schemas-xmlsoap-org:ws:2005:04:discovery</a:To>
	</s:Header>
	<s:Body>
		<Probe xmlns="http://schemas.xmlsoap.org/ws/2005/04/discovery" xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery" xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
			<d:Types>dn:NetworkVideoTransmitter</d:Types>
		</Probe>
	</s:Body>
</s:Envelope>`, uuid)
}

func (dd *DeviceDiscovery) sendUDPMulticast(msg string, interfaceName string) []string {
	var result []string

	if interfaceName == "" {
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil
		}

		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 {
				responses := dd.sendMulticastOnInterface(msg, &iface)
				result = append(result, responses...)
			}
		}
	} else {
		iface, err := net.InterfaceByName(interfaceName)
		if err != nil {
			return nil
		}
		result = dd.sendMulticastOnInterface(msg, iface)
	}

	return result
}

func (dd *DeviceDiscovery) sendMulticastOnInterface(msg string, iface *net.Interface) []string {
	var result []string

	c, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return nil
	}
	defer c.Close()

	p := ipv4.NewPacketConn(c)
	group := net.ParseIP(wsDiscoveryMulticastIP)

	if err := p.JoinGroup(iface, &net.UDPAddr{IP: group}); err != nil {
		return result
	}

	dst := &net.UDPAddr{IP: group, Port: wsDiscoveryPort}

	if err := p.SetMulticastInterface(iface); err != nil {
		return result
	}

	p.SetMulticastTTL(wsDiscoveryMulticastTTL)

	if _, err := p.WriteTo([]byte(msg), nil, dst); err != nil {
		return result
	}

	if err := p.SetReadDeadline(time.Now().Add(wsDiscoveryTimeout)); err != nil {
		return result
	}

	for {
		b := make([]byte, bufSize)
		n, _, _, err := p.ReadFrom(b)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				// 超时是正常的
			}
			break
		}
		result = append(result, string(b[0:n]))
	}

	return result
}

func (dd *DeviceDiscovery) parseProbeResponse(response string) []string {
	var xaddrs []string

	re := regexp.MustCompile(`<.*?XAddrs.*?>(.*?)</.*?XAddrs.*?>`)
	matches := re.FindStringSubmatch(response)

	if len(matches) > 1 {
		text := matches[1]
		urls := strings.Split(text, " ")
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url != "" {
				xaddrs = append(xaddrs, url)
			}
		}
	}

	return xaddrs
}

func (dd *DeviceDiscovery) probeDevice(xaddr string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}

	soap := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Body>
		<tds:GetDeviceInformation/>
	</s:Body>
</s:Envelope>`

	req, err := http.NewRequest("POST", xaddr, strings.NewReader(soap))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 401 表示需要认证,也是ONVIF设备
	if resp.StatusCode == http.StatusUnauthorized {
		return true
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	bodyStr := string(body)

	return strings.Contains(bodyStr, "Envelope") &&
		(strings.Contains(bodyStr, "onvif") ||
			strings.Contains(bodyStr, "GetDeviceInformationResponse") ||
			strings.Contains(bodyStr, "Manufacturer"))
}

func (dd *DeviceDiscovery) generateIPRange(startIP, endIP string) ([]string, error) {
	start := net.ParseIP(startIP)
	end := net.ParseIP(endIP)

	if start == nil || end == nil {
		return nil, errors.New("无效的IP地址")
	}

	start = start.To4()
	end = end.To4()

	if start == nil || end == nil {
		return nil, errors.New("仅支持IPv4地址")
	}

	startInt := ipToUint32(start)
	endInt := ipToUint32(end)

	if startInt > endInt {
		return nil, errors.New("起始IP必须小于或等于结束IP")
	}

	var ips []string
	for i := startInt; i <= endInt; i++ {
		ips = append(ips, uint32ToIP(i).String())
	}

	return ips, nil
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

func (dd *DeviceDiscovery) parseDeviceInfo(xaddr string) ONVIFDevice {
	device := ONVIFDevice{XAddr: xaddr}

	// 解析URL提取IP和端口
	if strings.HasPrefix(xaddr, "http://") {
		xaddr = strings.TrimPrefix(xaddr, "http://")
	}

	parts := strings.Split(xaddr, "/")
	if len(parts) > 0 {
		hostPort := parts[0]
		hostParts := strings.Split(hostPort, ":")

		device.IP = hostParts[0]

		if len(hostParts) > 1 {
			fmt.Sscanf(hostParts[1], "%d", &device.Port)
		} else {
			device.Port = 80
		}

		if len(parts) > 1 {
			device.Path = "/" + strings.Join(parts[1:], "/")
		}
	}

	return device
}
