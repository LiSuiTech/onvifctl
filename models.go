package main

import "encoding/xml"

// SOAP 信封结构
type Envelope struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Header  *Header  `xml:"Header,omitempty"`
	Body    Body     `xml:"Body"`
}

type Header struct {
	Security Security `xml:"Security"`
}

type Security struct {
	XMLName        xml.Name `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd Security"`
	UsernameToken  UsernameToken
	MustUnderstand string `xml:"http://www.w3.org/2003/05/soap-envelope mustUnderstand,attr"`
}

type UsernameToken struct {
	Username string   `xml:"Username"`
	Password Password `xml:"Password"`
	Nonce    string   `xml:"Nonce"`
	Created  string   `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Created"`
}

type Password struct {
	Type  string `xml:"Type,attr"`
	Value string `xml:",chardata"`
}

type Body struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
	Content interface{}
}

// 设备信息请求/响应
type GetDeviceInformation struct {
	XMLName xml.Name `xml:"http://www.onvif.org/ver10/device/wsdl GetDeviceInformation"`
}

type GetDeviceInformationResponse struct {
	Manufacturer    string `xml:"Manufacturer"`
	Model           string `xml:"Model"`
	FirmwareVersion string `xml:"FirmwareVersion"`
	SerialNumber    string `xml:"SerialNumber"`
	HardwareId      string `xml:"HardwareId"`
}

// 系统时间
type GetSystemDateAndTime struct {
	XMLName xml.Name `xml:"http://www.onvif.org/ver10/device/wsdl GetSystemDateAndTime"`
}

type GetSystemDateAndTimeResponse struct {
	SystemDateAndTime SystemDateAndTime `xml:"SystemDateAndTime"`
}

type SystemDateAndTime struct {
	DateTimeType string      `xml:"DateTimeType"`
	UTCDateTime  UTCDateTime `xml:"UTCDateTime"`
}

type UTCDateTime struct {
	Time Time `xml:"Time"`
	Date Date `xml:"Date"`
}

type Time struct {
	Hour   int `xml:"Hour"`
	Minute int `xml:"Minute"`
	Second int `xml:"Second"`
}

type Date struct {
	Year  int `xml:"Year"`
	Month int `xml:"Month"`
	Day   int `xml:"Day"`
}

// Media 配置
type GetProfiles struct {
	XMLName xml.Name `xml:"http://www.onvif.org/ver10/media/wsdl GetProfiles"`
}

type GetProfilesResponse struct {
	Profiles []Profile `xml:"Profiles"`
}

type Profile struct {
	Token string `xml:"token,attr"`
	Name  string `xml:"Name"`
}

// 流 URI
type GetStreamUri struct {
	XMLName      xml.Name    `xml:"http://www.onvif.org/ver10/media/wsdl GetStreamUri"`
	StreamSetup  StreamSetup `xml:"StreamSetup"`
	ProfileToken string      `xml:"ProfileToken"`
}

type StreamSetup struct {
	Stream    string    `xml:"Stream"`
	Transport Transport `xml:"Transport"`
}

type Transport struct {
	Protocol string `xml:"Protocol"`
}

type GetStreamUriResponse struct {
	MediaUri MediaUri `xml:"MediaUri"`
}

type MediaUri struct {
	Uri string `xml:"Uri"`
}

// PTZ 相关结构
type ContinuousMove struct {
	XMLName      xml.Name `xml:"http://www.onvif.org/ver20/ptz/wsdl ContinuousMove"`
	ProfileToken string   `xml:"ProfileToken"`
	Velocity     PTZSpeed `xml:"Velocity"`
	Timeout      string   `xml:"Timeout,omitempty"`
}

type PTZSpeed struct {
	PanTilt PanTilt `xml:"PanTilt"`
	Zoom    Zoom    `xml:"Zoom"`
}

type PanTilt struct {
	X     float64 `xml:"x,attr"`
	Y     float64 `xml:"y,attr"`
	Space string  `xml:"space,attr,omitempty"`
}

type Zoom struct {
	X     float64 `xml:"x,attr"`
	Space string  `xml:"space,attr,omitempty"`
}

type Stop struct {
	XMLName      xml.Name `xml:"http://www.onvif.org/ver20/ptz/wsdl Stop"`
	ProfileToken string   `xml:"ProfileToken"`
	PanTilt      bool     `xml:"PanTilt,omitempty"`
	Zoom         bool     `xml:"Zoom,omitempty"`
}

type GotoPreset struct {
	XMLName      xml.Name `xml:"http://www.onvif.org/ver20/ptz/wsdl GotoPreset"`
	ProfileToken string   `xml:"ProfileToken"`
	PresetToken  string   `xml:"PresetToken"`
}

type SetPreset struct {
	XMLName      xml.Name `xml:"http://www.onvif.org/ver20/ptz/wsdl SetPreset"`
	ProfileToken string   `xml:"ProfileToken"`
	PresetName   string   `xml:"PresetName,omitempty"`
}

type SetPresetResponse struct {
	PresetToken string `xml:"PresetToken"`
}

type GetPresets struct {
	XMLName      xml.Name `xml:"http://www.onvif.org/ver20/ptz/wsdl GetPresets"`
	ProfileToken string   `xml:"ProfileToken"`
}

type GetPresetsResponse struct {
	Presets []PTZPreset `xml:"Preset"`
}

type PTZPreset struct {
	Token    string      `xml:"token,attr"`
	Name     string      `xml:"Name"`
	Position PTZPosition `xml:"PTZPosition,omitempty"`
}

type PTZPosition struct {
	PanTilt PanTilt `xml:"PanTilt"`
	Zoom    Zoom    `xml:"Zoom"`
}

// 抓图相关
type GetSnapshotUri struct {
	XMLName      xml.Name `xml:"http://www.onvif.org/ver10/media/wsdl GetSnapshotUri"`
	ProfileToken string   `xml:"ProfileToken"`
}

type GetSnapshotUriResponse struct {
	MediaUri MediaUri `xml:"MediaUri"`
}

// 视频编码配置
type GetVideoEncoderConfigurations struct {
	XMLName xml.Name `xml:"http://www.onvif.org/ver10/media/wsdl GetVideoEncoderConfigurations"`
}

type GetVideoEncoderConfigurationsResponse struct {
	Configurations []VideoEncoderConfiguration `xml:"Configurations"`
}

type VideoEncoderConfiguration struct {
	Token       string      `xml:"token,attr"`
	Name        string      `xml:"Name"`
	Encoding    string      `xml:"Encoding"`
	Resolution  Resolution  `xml:"Resolution"`
	Quality     float64     `xml:"Quality"`
	RateControl RateControl `xml:"RateControl"`
}

type Resolution struct {
	Width  int `xml:"Width"`
	Height int `xml:"Height"`
}

type RateControl struct {
	FrameRateLimit   int `xml:"FrameRateLimit"`
	EncodingInterval int `xml:"EncodingInterval"`
	BitrateLimit     int `xml:"BitrateLimit"`
}

type SetVideoEncoderConfiguration struct {
	XMLName          xml.Name                  `xml:"http://www.onvif.org/ver10/media/wsdl SetVideoEncoderConfiguration"`
	Configuration    VideoEncoderConfiguration `xml:"Configuration"`
	ForcePersistence bool                      `xml:"ForcePersistence"`
}

// 网络配置
type GetNetworkInterfaces struct {
	XMLName xml.Name `xml:"http://www.onvif.org/ver10/device/wsdl GetNetworkInterfaces"`
}

type GetNetworkInterfacesResponse struct {
	NetworkInterfaces []NetworkInterface `xml:"NetworkInterfaces"`
}

type NetworkInterface struct {
	Token   string               `xml:"token,attr"`
	Enabled bool                 `xml:"Enabled"`
	Info    NetworkInterfaceInfo `xml:"Info"`
	IPv4    IPv4Configuration    `xml:"IPv4"`
}

type NetworkInterfaceInfo struct {
	Name      string `xml:"Name"`
	HwAddress string `xml:"HwAddress"`
	MTU       int    `xml:"MTU"`
}

type IPv4Configuration struct {
	Enabled bool                  `xml:"Enabled"`
	DHCP    bool                  `xml:"DHCP"`
	Manual  []PrefixedIPv4Address `xml:"Manual"`
}

type PrefixedIPv4Address struct {
	Address      string `xml:"Address"`
	PrefixLength int    `xml:"PrefixLength"`
}
