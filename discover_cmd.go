package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"onvifctl/discovery" // 根据你的项目路径调整
)

// DiscoveredDevice 发现的设备信息
type DiscoveredDevice struct {
	IP           string
	Port         int
	XAddr        string
	Manufacturer string
	Model        string
	FirmwareVer  string
	SerialNumber string
	AuthType     string // ws-security / digest / basic / none / unknown
	AuthResult   string // 认证结果: success / failed / untested
	Reachable    bool
}

func discoverCmd() *cobra.Command {
	var (
		mode          string // broadcast, ip, subnet
		interfaceName string
		ipAddress     string
		subnet        string
		startIP       string
		endIP         string
		ports         []int
		timeout       int
		credentials   []string // username:password 格式
		saveFile      string
		jsonOutput    bool
		verbose       bool
	)

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "设备发现",
		Long: `在局域网内发现 ONVIF 设备
支持三种模式:
  - broadcast: 广播发现(默认)
  - ip: 扫描指定 IP 地址
  - subnet: 扫描指定网段`,
		Example: `  # 广播发现所有设备
  onvifctl discover

  # 指定网卡进行广播发现
  onvifctl discover --interface eth0

  # 扫描单个 IP
  onvifctl discover --mode ip --ip 192.168.1.100

  # 扫描 IP 范围
  onvifctl discover --mode subnet --start 192.168.1.1 --end 192.168.1.254

  # 扫描指定网段
  onvifctl discover --mode subnet --subnet 192.168.1.0/24

  # 指定多个用户名密码
  onvifctl discover --cred admin:admin --cred admin:12345

  # 自定义端口扫描
  onvifctl discover --mode subnet --subnet 192.168.1.0/24 --ports 80,8080,8000

  # 保存结果到文件
  onvifctl discover --save devices.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 解析凭据
			creds := parseCredentials(credentials)
			if len(creds) == 0 {
				// 默认凭据
				creds = []discovery.Credential{
					{Username: "admin", Password: "admin"},
					{Username: "admin", Password: "12345"},
					{Username: "admin", Password: ""},
					{Username: "root", Password: "root"},
				}
			}

			var devices []discovery.ONVIFDevice
			var err error

			dd := discovery.NewDeviceDiscovery()

			switch mode {
			case "broadcast":
				fmt.Println("=== 广播发现模式 ===")
				devices, err = dd.DiscoverByBroadcast(interfaceName)
				if err != nil {
					return fmt.Errorf("广播发现失败: %w", err)
				}

			case "ip":
				if ipAddress == "" {
					return fmt.Errorf("必须指定 IP 地址 (--ip)")
				}
				fmt.Printf("=== 扫描单个 IP: %s ===\n", ipAddress)
				devices, err = dd.DiscoverByIPRange(ipAddress, ipAddress, ports, time.Duration(timeout)*time.Second)
				if err != nil {
					return fmt.Errorf("IP 扫描失败: %w", err)
				}

			case "subnet":
				if subnet != "" {
					// 解析 CIDR
					start, end, err := parseCIDR(subnet)
					if err != nil {
						return fmt.Errorf("无效的子网: %w", err)
					}
					startIP = start
					endIP = end
				}

				if startIP == "" || endIP == "" {
					return fmt.Errorf("必须指定 IP 范围 (--start --end 或 --subnet)")
				}

				fmt.Printf("=== 扫描网段: %s - %s ===\n", startIP, endIP)
				devices, err = dd.DiscoverByIPRange(startIP, endIP, ports, time.Duration(timeout)*time.Second)
				if err != nil {
					return fmt.Errorf("网段扫描失败: %w", err)
				}

			default:
				return fmt.Errorf("未知的模式: %s (支持: broadcast, ip, subnet)", mode)
			}

			if len(devices) == 0 {
				fmt.Println("\n未发现任何 ONVIF 设备")
				return nil
			}

			// 获取设备详细信息
			fmt.Printf("\n正在获取设备详细信息...\n")
			detailedDevices := getDeviceDetails(devices, creds, verbose)

			// 显示结果
			if jsonOutput {
				printDevicesJSON(detailedDevices)
			} else {
				printDevicesTable(detailedDevices)
			}

			// 保存到文件
			if saveFile != "" {
				if err := saveDevicesToFile(detailedDevices, saveFile); err != nil {
					return fmt.Errorf("保存设备列表失败: %w", err)
				}
				fmt.Printf("\n✓ 设备列表已保存到: %s\n", saveFile)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&mode, "mode", "m", "broadcast", "发现模式: broadcast, ip, subnet")
	cmd.Flags().StringVarP(&interfaceName, "interface", "i", "", "网络接口名称(仅 broadcast 模式)")
	cmd.Flags().StringVar(&ipAddress, "ip", "", "目标 IP 地址(仅 ip 模式)")
	cmd.Flags().StringVar(&subnet, "subnet", "", "目标子网 CIDR 格式(如 192.168.1.0/24)")
	cmd.Flags().StringVar(&startIP, "start", "", "起始 IP 地址")
	cmd.Flags().StringVar(&endIP, "end", "", "结束 IP 地址")
	cmd.Flags().IntSliceVar(&ports, "ports", []int{80, 8080, 8000, 8899, 554}, "扫描端口列表")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 2, "连接超时时间(秒)")
	cmd.Flags().StringArrayVarP(&credentials, "cred", "c", []string{}, "认证凭据 username:password (可多次指定)")
	cmd.Flags().StringVarP(&saveFile, "save", "o", "", "保存设备列表到文件")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "以 JSON 格式输出")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示详细的认证过程")

	return cmd
}

// parseCredentials 解析用户名密码
func parseCredentials(credStrings []string) []discovery.Credential {
	var creds []discovery.Credential
	for _, credStr := range credStrings {
		parts := strings.SplitN(credStr, ":", 2)
		if len(parts) == 2 {
			creds = append(creds, discovery.Credential{
				Username: parts[0],
				Password: parts[1],
			})
		}
	}
	return creds
}

// parseCIDR 解析 CIDR 格式的子网
func parseCIDR(cidr string) (string, string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", err
	}

	// 获取网络地址和广播地址
	startIP := ipnet.IP
	endIP := make(net.IP, len(startIP))
	copy(endIP, startIP)

	// 计算广播地址
	for i := range endIP {
		endIP[i] |= ^ipnet.Mask[i]
	}

	// 跳过网络地址和广播地址
	startIP[3]++
	endIP[3]--

	return startIP.String(), endIP.String(), nil
}

// getDeviceDetails 获取设备详细信息
func getDeviceDetails(devices []discovery.ONVIFDevice, creds []discovery.Credential, verbose bool) []DiscoveredDevice {
	var result []DiscoveredDevice
	var mu sync.Mutex
	var wg sync.WaitGroup

	dim := discovery.NewDeviceInfoManager(creds)

	for _, device := range devices {
		wg.Add(1)
		go func(dev discovery.ONVIFDevice) {
			defer wg.Done()

			discovered := DiscoveredDevice{
				IP:         dev.IP,
				Port:       dev.Port,
				XAddr:      dev.XAddr,
				Reachable:  true,
				AuthResult: "untested",
			}

			// 尝试获取设备信息
			//if !verbose {
			//	fmt.Printf("\n[%s] 正在获取设备信息...\n", dev.IP)
			//}

			// 使用增强的认证方法
			info, err := dim.GetDeviceInfoEnhanced(dev.XAddr, verbose)
			if err != nil {
				discovered.AuthType = "unknown"
				discovered.AuthResult = "failed"

				//if !verbose {
				//	fmt.Printf("[%s] ✗ 认证失败: %v\n", dev.IP, err)
				//}

				// 判断失败原因
				if strings.Contains(err.Error(), "401") ||
					strings.Contains(err.Error(), "Unauthorized") ||
					strings.Contains(err.Error(), "NotAuthorized") {
					discovered.AuthType = "auth-required"
					//if !verbose {
					//	fmt.Printf("[%s] ℹ 设备需要认证(请检查用户名密码)\n", dev.IP)
					//}
				} else if strings.Contains(err.Error(), "所有凭据都认证失败") {
					discovered.AuthResult = "failed-all-creds"
					//if !verbose {
					//	fmt.Printf("[%s] ℹ 所有提供的凭据都认证失败\n", dev.IP)
					//}
				}
			} else {
				discovered.Manufacturer = info.Manufacturer
				discovered.Model = info.Model
				discovered.FirmwareVer = info.FirmwareVer
				discovered.SerialNumber = info.SerialNumber
				discovered.AuthType = info.AuthType
				discovered.AuthResult = "success"

				//if !verbose {
				//	fmt.Printf("[%s] ✓ 认证成功 [%s]\n", dev.IP, strings.ToUpper(info.AuthType))
				//	fmt.Printf("[%s] ✓ 厂商: %s, 型号: %s\n",
				//		dev.IP, info.Manufacturer, info.Model)
				//}
			}

			mu.Lock()
			result = append(result, discovered)
			mu.Unlock()
		}(device)
	}

	wg.Wait()
	return result
}

// printDevicesTable 表格形式打印设备列表（居中对齐格式）
func printDevicesTable(devices []DiscoveredDevice) {
	fmt.Printf("\n发现 %d 个 ONVIF 设备:\n\n", len(devices))

	// 打印表头（居中）
	printCentered("序号", 6)
	printCentered("IP地址", 17)
	printCentered("端口", 8)
	printCentered("厂商", 14)
	printCentered("型号", 22)
	printCentered("固件版本", 24)
	printCentered("序列号", 42)
	printCentered("认证方式", 14)
	fmt.Println("认证结果")

	fmt.Println(strings.Repeat("-", 155))

	for i, device := range devices {
		manufacturer := device.Manufacturer
		if manufacturer == "" {
			manufacturer = "-"
		}
		model := device.Model
		if model == "" {
			model = "-"
		}
		firmware := device.FirmwareVer
		if firmware == "" {
			firmware = "-"
		}
		serial := device.SerialNumber
		if serial == "" {
			serial = "-"
		}

		authType := device.AuthType
		if authType == "" {
			authType = "unknown"
		}

		// 格式化认证结果
		authResult := formatAuthResult(device.AuthResult, device.AuthType)

		// 打印每行数据（居中）
		printCentered(fmt.Sprintf("%d", i+1), 6)
		printCentered(device.IP, 17)
		printCentered(fmt.Sprintf("%d", device.Port), 8)
		printCentered(manufacturer, 14)
		printCentered(model, 22)
		printCentered(firmware, 24)
		printCentered(serial, 42)
		printCentered(authType, 14)
		fmt.Println(authResult)
	}
}

// printCentered 居中打印字符串
func printCentered(s string, width int) {
	// 计算字符串的显示宽度（中文字符占2个宽度）
	displayWidth := 0
	for _, r := range s {
		if r > 127 {
			displayWidth += 2
		} else {
			displayWidth += 1
		}
	}

	// 计算左右需要的空格数
	totalPadding := width - displayWidth
	if totalPadding < 0 {
		totalPadding = 0
	}

	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding

	fmt.Print(strings.Repeat(" ", leftPadding))
	fmt.Print(s)
	fmt.Print(strings.Repeat(" ", rightPadding))
}

// formatAuthResult 格式化认证结果显示
func formatAuthResult(result, authType string) string {
	switch result {
	case "success":
		// 成功时显示使用的认证方式
		switch authType {
		case "wsse":
			return "✓ 成功(WS-Security)"
		case "digest":
			return "✓ 成功(Digest)"
		case "basic":
			return "✓ 成功(Basic)"
		case "none":
			return "✓ 成功(无认证)"
		default:
			return "✓ 成功"
		}
	case "failed":
		return "✗ 失败"
	case "failed-all-creds":
		return "✗ 凭据错误"
	case "untested":
		return "- 未测试"
	default:
		return result
	}
}

// printDevicesJSON JSON 格式打印设备列表
func printDevicesJSON(devices []DiscoveredDevice) {
	fmt.Println("[")
	for i, device := range devices {
		fmt.Printf("  {\n")
		fmt.Printf("    \"ip\": \"%s\",\n", device.IP)
		fmt.Printf("    \"port\": %d,\n", device.Port)
		fmt.Printf("    \"xaddr\": \"%s\",\n", device.XAddr)
		fmt.Printf("    \"manufacturer\": \"%s\",\n", device.Manufacturer)
		fmt.Printf("    \"model\": \"%s\",\n", device.Model)
		fmt.Printf("    \"firmware\": \"%s\",\n", device.FirmwareVer)
		fmt.Printf("    \"serial\": \"%s\",\n", device.SerialNumber)
		fmt.Printf("    \"authType\": \"%s\",\n", device.AuthType)
		fmt.Printf("    \"authResult\": \"%s\"\n", device.AuthResult)
		if i < len(devices)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}

// saveDevicesToFile 保存设备列表到文件
func saveDevicesToFile(devices []DiscoveredDevice, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入表头
	fmt.Fprintln(file, "# ONVIF 设备列表")
	fmt.Fprintf(file, "# 发现时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "# 设备总数: %d\n\n", len(devices))

	w := tabwriter.NewWriter(file, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IP 地址\t端口\t厂商\t型号\t固件版本\t序列号\t认证方式\t认证结果\tXAddr")
	fmt.Fprintln(w, "-------\t----\t----\t----\t--------\t------\t--------\t--------\t-----")

	for _, device := range devices {
		manufacturer := device.Manufacturer
		if manufacturer == "" {
			manufacturer = "-"
		}
		model := device.Model
		if model == "" {
			model = "-"
		}
		firmware := device.FirmwareVer
		if firmware == "" {
			firmware = "-"
		}
		serial := device.SerialNumber
		if serial == "" {
			serial = "-"
		}

		authResult := formatAuthResult(device.AuthResult, device.AuthType)

		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			device.IP,
			device.Port,
			manufacturer,
			model,
			firmware,
			serial,
			device.AuthType,
			authResult,
			device.XAddr,
		)
	}

	w.Flush()
	return nil
}
