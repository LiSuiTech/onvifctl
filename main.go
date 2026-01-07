package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	host     string
	port     int
	username string
	password string
	debug    bool
	authMode string
	useHTTPS bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "onvifctl",
		Short:   "ONVIF Command Line Tool",
		Long:    "ONVIF 协议命令行工具，用于管理和控制支持 ONVIF 的网络摄像头设备",
		Version: "1.0.0",
	}

	// 全局 flags
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "", "设备 IP 地址或主机名")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "P", 80, "设备端口号")
	rootCmd.PersistentFlags().StringVarP(&username, "user", "u", "admin", "ONVIF 登录用户名")
	rootCmd.PersistentFlags().StringVarP(&password, "pass", "w", "", "ONVIF 登录密码")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "启用调试日志")
	rootCmd.PersistentFlags().StringVarP(&authMode, "auth", "a", "ws-security", "认证模式: ws-security 或 digest")
	rootCmd.PersistentFlags().BoolVarP(&useHTTPS, "https", "s", false, "使用 HTTPS 协议")

	// 添加子命令
	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(streamCmd())
	rootCmd.AddCommand(ptzCmd())
	rootCmd.AddCommand(snapshotCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(discoverCmd()) // 新的增强版 discover 命令
	rootCmd.AddCommand(timeCmd())
	rootCmd.AddCommand(eventsCmd())
	rootCmd.AddCommand(batchCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func infoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "获取设备信息",
		Long:  "获取设备信息（厂商、型号、时间、能力等）",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}
			if port < 1 || port > 65535 {
				return fmt.Errorf("端口号必须在 1-65535 之间")
			}
			if authMode != "ws-security" && authMode != "digest" {
				return fmt.Errorf("认证模式必须是 ws-security 或 digest")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.GetDeviceInfo()
		},
	}

	return cmd
}

func streamCmd() *cobra.Command {
	var profile int

	cmd := &cobra.Command{
		Use:   "stream",
		Short: "获取 RTSP 流地址",
		Long:  "获取设备的 RTSP 视频流地址",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}
			if port < 1 || port > 65535 {
				return fmt.Errorf("端口号必须在 1-65535 之间")
			}
			if authMode != "ws-security" && authMode != "digest" {
				return fmt.Errorf("认证模式必须是 ws-security 或 digest")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.GetStreamURI(profile)
		},
	}

	cmd.Flags().IntVarP(&profile, "profile", "r", 0, "配置文件索引（0 表示主码流）")

	return cmd
}

func ptzCmd() *cobra.Command {
	var (
		panSpeed  float64
		tiltSpeed float64
		zoomSpeed float64
		timeout   int
		preset    int
		action    string
	)

	cmd := &cobra.Command{
		Use:   "ptz",
		Short: "PTZ 云台控制",
		Long:  "控制摄像头云台移动、缩放、预置位等",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}
			if port < 1 || port > 65535 {
				return fmt.Errorf("端口号必须在 1-65535 之间")
			}
			if authMode != "ws-security" && authMode != "digest" {
				return fmt.Errorf("认证模式必须是 ws-security 或 digest")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			switch action {
			case "move":
				return client.PTZMove(panSpeed, tiltSpeed, zoomSpeed, timeout)
			case "stop":
				return client.PTZStop()
			case "goto":
				if preset == 0 {
					return fmt.Errorf("必须指定预置位编号 (--preset)")
				}
				return client.PTZGotoPreset(preset)
			case "setpreset":
				if preset == 0 {
					return fmt.Errorf("必须指定预置位编号 (--preset)")
				}
				return client.PTZSetPreset(preset)
			case "list":
				return client.PTZListPresets()
			default:
				return fmt.Errorf("未知的操作: %s (支持: move, stop, goto, setpreset, list)", action)
			}
		},
	}

	cmd.Flags().StringVar(&action, "action", "", "操作类型: move, stop, goto, setpreset, list (必填)")
	cmd.Flags().Float64Var(&panSpeed, "pan", 0, "水平速度 (-1.0 到 1.0, 负值向左)")
	cmd.Flags().Float64Var(&tiltSpeed, "tilt", 0, "垂直速度 (-1.0 到 1.0, 负值向下)")
	cmd.Flags().Float64Var(&zoomSpeed, "zoom", 0, "缩放速度 (-1.0 到 1.0, 负值缩小)")
	cmd.Flags().IntVar(&timeout, "timeout", 1, "移动持续时间（秒）")
	cmd.Flags().IntVar(&preset, "preset", 0, "预置位编号")
	cmd.MarkFlagRequired("action")

	return cmd
}

func snapshotCmd() *cobra.Command {
	var (
		output  string
		profile int
	)

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "抓取图像",
		Long:  "从摄像头抓取当前画面并保存为 JPEG 图像",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}
			if port < 1 || port > 65535 {
				return fmt.Errorf("端口号必须在 1-65535 之间")
			}
			if authMode != "ws-security" && authMode != "digest" {
				return fmt.Errorf("认证模式必须是 ws-security 或 digest")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.GetSnapshot(output, profile)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "snapshot.jpg", "输出文件路径")
	cmd.Flags().IntVarP(&profile, "profile", "r", 0, "配置文件索引")

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "配置管理",
		Long:  "查看和修改设备配置",
	}

	// 子命令: 获取视频编码配置
	getVideoCmd := &cobra.Command{
		Use:   "get-video",
		Short: "获取视频编码配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.GetVideoEncoderConfiguration()
		},
	}

	// 子命令: 设置视频编码配置
	setVideoCmd := &cobra.Command{
		Use:   "set-video",
		Short: "设置视频编码配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			width, _ := cmd.Flags().GetInt("width")
			height, _ := cmd.Flags().GetInt("height")
			fps, _ := cmd.Flags().GetInt("fps")
			bitrate, _ := cmd.Flags().GetInt("bitrate")

			return client.SetVideoEncoderConfiguration(width, height, fps, bitrate)
		},
	}

	setVideoCmd.Flags().Int("width", 0, "视频宽度")
	setVideoCmd.Flags().Int("height", 0, "视频高度")
	setVideoCmd.Flags().Int("fps", 0, "帧率")
	setVideoCmd.Flags().Int("bitrate", 0, "比特率 (kbps)")

	// 子命令: 获取网络配置
	getNetworkCmd := &cobra.Command{
		Use:   "get-network",
		Short: "获取网络配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.GetNetworkConfiguration()
		},
	}

	cmd.AddCommand(getVideoCmd)
	cmd.AddCommand(setVideoCmd)
	cmd.AddCommand(getNetworkCmd)

	return cmd
}

func timeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time",
		Short: "时间管理",
		Long:  "查看和同步设备时间",
	}

	// 子命令: 获取时间
	getTimeCmd := &cobra.Command{
		Use:   "get",
		Short: "获取设备时间",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.GetSystemTime()
		},
	}

	// 子命令: 同步时间
	syncTimeCmd := &cobra.Command{
		Use:   "sync",
		Short: "同步设备时间到系统时间",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.SyncSystemTime()
		},
	}

	// 子命令: 设置 NTP
	setNTPCmd := &cobra.Command{
		Use:   "set-ntp",
		Short: "设置 NTP 服务器",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			ntpServer, _ := cmd.Flags().GetString("server")
			if ntpServer == "" {
				return fmt.Errorf("必须指定 NTP 服务器地址 (--server)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.SetNTP(ntpServer)
		},
	}

	setNTPCmd.Flags().String("server", "", "NTP 服务器地址")

	cmd.AddCommand(getTimeCmd)
	cmd.AddCommand(syncTimeCmd)
	cmd.AddCommand(setNTPCmd)

	return cmd
}

func eventsCmd() *cobra.Command {
	var (
		duration int
		filter   string
	)

	cmd := &cobra.Command{
		Use:   "events",
		Short: "事件订阅",
		Long:  "订阅和监听设备事件（移动侦测、报警等）",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			client.AuthMode = authMode

			return client.SubscribeEvents(duration, filter)
		},
	}

	cmd.Flags().IntVarP(&duration, "duration", "t", 60, "订阅持续时间（秒）")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "事件过滤器（留空订阅所有事件）")

	return cmd
}

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "批量设备管理",
		Long:  "对多个设备执行批量操作",
	}

	// 子命令: 导入配置
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "从配置文件导入设备列表",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			if configFile == "" {
				return fmt.Errorf("必须指定配置文件 (--file)")
			}

			config, err := LoadBatchConfig(configFile)
			if err != nil {
				return fmt.Errorf("加载配置文件失败: %w", err)
			}

			fmt.Printf("✓ 成功加载 %d 个设备配置\n", len(config.Devices))
			for i, device := range config.Devices {
				fmt.Printf("  [%d] %s - %s:%d\n", i+1, device.Name, device.Host, device.Port)
			}

			return nil
		},
	}

	importCmd.Flags().String("file", "devices.yaml", "配置文件路径")

	// 子命令: 导出配置
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "导出设备配置到文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			if configFile == "" {
				return fmt.Errorf("必须指定配置文件 (--file)")
			}

			// 创建示例配置
			config := &BatchConfig{
				Devices: []DeviceConfig{
					{
						Name:     "Camera-01",
						Host:     "192.168.1.100",
						Port:     80,
						Username: "admin",
						Password: "12345",
						UseHTTPS: false,
					},
					{
						Name:     "Camera-02",
						Host:     "192.168.1.101",
						Port:     80,
						Username: "admin",
						Password: "12345",
						UseHTTPS: false,
					},
				},
			}

			if err := SaveBatchConfig(config, configFile); err != nil {
				return fmt.Errorf("导出配置失败: %w", err)
			}

			fmt.Printf("✓ 配置已导出到: %s\n", configFile)
			fmt.Println("  请编辑该文件以添加你的设备信息")

			return nil
		},
	}

	exportCmd.Flags().String("file", "devices.yaml", "配置文件路径")

	// 子命令: 批量获取信息
	infoAllCmd := &cobra.Command{
		Use:   "info",
		Short: "批量获取设备信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			if configFile == "" {
				return fmt.Errorf("必须指定配置文件 (--file)")
			}

			config, err := LoadBatchConfig(configFile)
			if err != nil {
				return fmt.Errorf("加载配置文件失败: %w", err)
			}

			return BatchGetInfo(config)
		},
	}

	infoAllCmd.Flags().String("file", "devices.yaml", "配置文件路径")

	// 子命令: 批量抓图
	snapshotAllCmd := &cobra.Command{
		Use:   "snapshot",
		Short: "批量抓取图像",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			outputDir, _ := cmd.Flags().GetString("output")
			if configFile == "" {
				return fmt.Errorf("必须指定配置文件 (--file)")
			}

			config, err := LoadBatchConfig(configFile)
			if err != nil {
				return fmt.Errorf("加载配置文件失败: %w", err)
			}

			return BatchSnapshot(config, outputDir)
		},
	}

	snapshotAllCmd.Flags().String("file", "devices.yaml", "配置文件路径")
	snapshotAllCmd.Flags().String("output", "snapshots", "输出目录")

	// 子命令: 批量同步时间
	syncAllCmd := &cobra.Command{
		Use:   "sync-time",
		Short: "批量同步设备时间",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			if configFile == "" {
				return fmt.Errorf("必须指定配置文件 (--file)")
			}

			config, err := LoadBatchConfig(configFile)
			if err != nil {
				return fmt.Errorf("加载配置文件失败: %w", err)
			}

			return BatchSyncTime(config)
		},
	}

	syncAllCmd.Flags().String("file", "devices.yaml", "配置文件路径")

	cmd.AddCommand(importCmd)
	cmd.AddCommand(exportCmd)
	cmd.AddCommand(infoAllCmd)
	cmd.AddCommand(snapshotAllCmd)
	cmd.AddCommand(syncAllCmd)

	return cmd
}
