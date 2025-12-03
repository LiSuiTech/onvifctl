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
		Use:   "onvifctl",
		Short: "ONVIF Command Line Tool",
		Long:  "ONVIF 协议命令行工具，用于管理和控制支持 ONVIF 的网络摄像头设备",
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

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}

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

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}

			return client.GetStreamURI(profile)
		},
	}

	cmd.Flags().IntVarP(&profile, "profile", "r", 0, "配置文件索引（0 表示主码流）")

	return cmd
}

func ptzCmd() *cobra.Command {
	var action string
	var pan, tilt, zoom float64
	var timeout, preset int

	cmd := &cobra.Command{
		Use:   "ptz",
		Short: "PTZ 控制",
		Long:  "PTZ 相关操作，如移动、定位等",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}
			if action == "move" {
				return client.PTZMove(pan, tilt, zoom, timeout)
			}
			if action == "setpreset" {
				return client.PTZSetPreset(preset)
			}
			if action == "goto" {
				return client.PTZGotoPreset(preset)
			}
			if action == "list" {
				return client.PTZListPresets()
			}
			if action == "stop" {
				return client.PTZStop()
			}
			return fmt.Errorf("未知的 PTZ 操作: %s", action)
		},
	}

	cmd.Flags().StringVar(&action, "action", "", "操作类型 (move, setpreset, goto, list, stop) (必填)")
	cmd.Flags().Float64Var(&pan, "pan", 0, "水平速度 (-1.0 到 1.0)")
	cmd.Flags().Float64Var(&tilt, "tilt", 0, "垂直速度 (-1.0 到 1.0)")
	cmd.Flags().Float64Var(&zoom, "zoom", 0, "缩放速度 (-1.0 到 1.0)")
	cmd.Flags().IntVar(&timeout, "timeout", 1, "移动持续时间(秒)")
	cmd.Flags().IntVar(&preset, "preset", 0, "预置位编号")
	
	// 标记 action 为必填参数
	cmd.MarkFlagRequired("action")

	return cmd
}

func snapshotCmd() *cobra.Command {
	var output string
	var profileIndex int

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "获取图片流",
		Long:  "获取设备的图片流",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}

			return client.GetSnapshot(output, profileIndex)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "snapshot.jpg", "输出文件路径")
	cmd.Flags().IntVarP(&profileIndex, "profile", "r", 0, "配置文件索引")

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "设备配置管理",
		Long:  "设备配置管理，包括视频编码配置和网络配置",
	}

	// get-video 子命令
	getVideoCmd := &cobra.Command{
		Use:   "get-video",
		Short: "获取视频编码配置",
		Long:  "获取设备的视频编码配置信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}

			return client.GetVideoEncoderConfiguration()
		},
	}

	// set-video 子命令
	var width, height, fps, bitrate int
	setVideoCmd := &cobra.Command{
		Use:   "set-video",
		Short: "设置视频编码配置",
		Long:  "设置设备的视频编码配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}

			return client.SetVideoEncoderConfiguration(width, height, fps, bitrate)
		},
	}
	setVideoCmd.Flags().IntVarP(&width, "width", "w", 0, "视频宽度")
	setVideoCmd.Flags().IntVarP(&height, "height", "h", 0, "视频高度")
	setVideoCmd.Flags().IntVarP(&fps, "fps", "f", 0, "帧率")
	setVideoCmd.Flags().IntVarP(&bitrate, "bitrate", "b", 0, "比特率 (kbps)")

	// get-network 子命令
	getNetworkCmd := &cobra.Command{
		Use:   "get-network",
		Short: "获取网络配置",
		Long:  "获取设备的网络配置信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("必须指定设备地址 (-H/--host)")
			}

			client, err := NewONVIFClient(host, port, username, password, debug, useHTTPS)
			if err != nil {
				return fmt.Errorf("连接设备失败: %w", err)
			}

			return client.GetNetworkConfiguration()
		},
	}

	cmd.AddCommand(getVideoCmd)
	cmd.AddCommand(setVideoCmd)
	cmd.AddCommand(getNetworkCmd)

	return cmd
}
