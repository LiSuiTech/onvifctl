package discovery

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// DeviceInfo 设备完整信息
type DeviceInfo struct {
	// 基本信息
	IPAddress string `json:"ipAddress"` // IP地址
	Port      int    `json:"port"`      // ONVIF端口
	XAddr     string `json:"xAddr"`     // ONVIF请求地址
	Protocol  string `json:"protocol"`  // ONVIF请求协议 (http/https)

	// 认证信息
	Username string `json:"username"` // 设备用户名
	Password string `json:"password"` // 设备密码
	AuthType string `json:"authType"` // 认证类型 (wsse/digest)

	// 设备信息
	DeviceName   string `json:"deviceName"`   // 设备名称
	Manufacturer string `json:"manufacturer"` // 厂商信息
	Model        string `json:"model"`        // 型号
	FirmwareVer  string `json:"firmwareVer"`  // 固件版本
	SerialNumber string `json:"serialNumber"` // 序列号
	HardwareId   string `json:"hardwareId"`   // 硬件ID

	// 通道信息
	Channels     []ChannelInfo `json:"channels"`     // 通道列表
	ChannelStart int           `json:"channelStart"` // 通道起始ID
	TotalChannel int           `json:"totalChannel"` // 总通道数

	// 媒体信息
	MediaXAddr       string   `json:"mediaXAddr"`       // 媒体服务地址
	MediaProtocol    string   `json:"mediaProtocol"`    // 媒体传输协议 (UDP/TCP)
	StreamProtocol   string   `json:"streamProtocol"`   // 码流协议 (RTSP/HTTP)
	SupportedProfile []string `json:"supportedProfile"` // 支持的配置文件

	// 能力信息
	Capabilities DeviceCapabilities `json:"capabilities"`
}

// ChannelInfo 通道信息
type ChannelInfo struct {
	ChannelNo   int    `json:"channelNo"`   // 通道号
	ChannelID   string `json:"channelId"`   // 通道ID
	ChannelName string `json:"channelName"` // 通道名称
	Token       string `json:"token"`       // Profile Token
	RTSPUrl     string `json:"rtspUrl"`     // RTSP地址
	Resolution  string `json:"resolution"`  // 分辨率
	VideoCodec  string `json:"videoCodec"`  // 视频编码
	AudioCodec  string `json:"audioCodec"`  // 音频编码
}

// DeviceCapabilities 设备能力
type DeviceCapabilities struct {
	DeviceService  string `json:"deviceService"`  // 设备服务地址
	MediaService   string `json:"mediaService"`   // 媒体服务地址
	EventService   string `json:"eventService"`   // 事件服务地址
	PTZService     string `json:"ptzService"`     // PTZ服务地址
	ImagingService string `json:"imagingService"` // 图像服务地址
}

// DeviceInfoManager 设备信息管理器
type DeviceInfoManager struct {
	client      *http.Client
	credentials []Credential
}

// Credential 认证凭据
type Credential struct {
	Username string
	Password string
}

// NewDeviceInfoManager 创建设备信息管理器
func NewDeviceInfoManager(credentials []Credential) *DeviceInfoManager {
	return &DeviceInfoManager{
		client:      &http.Client{Timeout: 10 * time.Second},
		credentials: credentials,
	}
}

// GetDeviceInfo 获取设备完整信息(自动尝试多组用户名密码)
func (dim *DeviceInfoManager) GetDeviceInfo(xaddr string) (*DeviceInfo, error) {
	//fmt.Printf("\n获取设备信息: %s\n", xaddr)

	// 尝试所有凭据
	for _, cred := range dim.credentials {
		//fmt.Printf("尝试凭据 [%d/%d]: %s\n", i+1, len(dim.credentials), cred.Username)

		info, err := dim.getDeviceInfoWithCredential(xaddr, cred.Username, cred.Password)
		if err == nil {
			//fmt.Printf("✓ 认证成功: %s/%s\n", cred.Username, cred.Password)
			return info, nil
		}

		//fmt.Printf("✗ 认证失败: %v\n", err)
	}

	return nil, fmt.Errorf("所有凭据都认证失败")
}

// GetDeviceInfoWithCredential 使用指定凭据获取设备信息
func (dim *DeviceInfoManager) GetDeviceInfoWithCredential(xaddr, username, password string) (*DeviceInfo, error) {
	return dim.getDeviceInfoWithCredential(xaddr, username, password)
}

// getDeviceInfoWithCredential 内部方法：使用指定凭据获取设备信息
func (dim *DeviceInfoManager) getDeviceInfoWithCredential(xaddr, username, password string) (*DeviceInfo, error) {
	info := &DeviceInfo{
		XAddr:    xaddr,
		Username: username,
		Password: password,
	}

	// 解析基本信息
	dim.parseBasicInfo(info)

	// 1. 尝试认证
	authType, err := dim.authenticate(info)
	if err != nil {
		return nil, fmt.Errorf("认证失败: %v", err)
	}
	info.AuthType = authType

	// 2. 获取设备信息
	dim.getDeviceInformation(info)

	// 3. 获取设备能力
	dim.getCapabilities(info)

	// 4. 获取通道信息
	dim.getChannels(info)

	return info, nil
}

// authenticate 尝试认证
func (dim *DeviceInfoManager) authenticate(info *DeviceInfo) (string, error) {
	// 先尝试 WS-Security (默认优先)
	soap := dim.buildGetDeviceInformationSOAP(info, "wsse")
	resp, err := dim.sendSOAPRequest(info.XAddr, soap)
	if err == nil && strings.Contains(resp, "GetDeviceInformationResponse") {
		return "wsse", nil
	}

	// 再尝试 Digest
	soap = dim.buildGetDeviceInformationSOAP(info, "digest")
	resp, err = dim.sendSOAPRequestWithDigest(info.XAddr, soap, info.Username, info.Password)
	if err == nil && strings.Contains(resp, "GetDeviceInformationResponse") {
		return "digest", nil
	}

	return "", errors.New("WS-Security 和 Digest 认证都失败")
}

// getDeviceInformation 获取设备基本信息
func (dim *DeviceInfoManager) getDeviceInformation(info *DeviceInfo) error {
	soap := dim.buildGetDeviceInformationSOAP(info, info.AuthType)

	var resp string
	var err error

	if info.AuthType == "digest" {
		resp, err = dim.sendSOAPRequestWithDigest(info.XAddr, soap, info.Username, info.Password)
	} else {
		resp, err = dim.sendSOAPRequest(info.XAddr, soap)
	}

	if err != nil {
		return err
	}

	// 使用 XML 解析
	type GetDeviceInformationResponse struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Response struct {
				Manufacturer string `xml:"Manufacturer"`
				Model        string `xml:"Model"`
				FirmwareVer  string `xml:"FirmwareVersion"`
				SerialNumber string `xml:"SerialNumber"`
				HardwareId   string `xml:"HardwareId"`
			} `xml:"GetDeviceInformationResponse"`
		} `xml:"Body"`
	}

	var deviceResp GetDeviceInformationResponse
	if err := xml.Unmarshal([]byte(resp), &deviceResp); err != nil {
		return fmt.Errorf("解析设备信息失败: %v", err)
	}

	info.Manufacturer = deviceResp.Body.Response.Manufacturer
	info.Model = deviceResp.Body.Response.Model
	info.FirmwareVer = deviceResp.Body.Response.FirmwareVer
	info.SerialNumber = deviceResp.Body.Response.SerialNumber
	info.HardwareId = deviceResp.Body.Response.HardwareId

	// 设备名称
	info.DeviceName = info.Model
	if info.DeviceName == "" {
		info.DeviceName = fmt.Sprintf("%s-%s", info.Manufacturer, info.SerialNumber)
	}

	return nil
}

// getCapabilities 获取设备能力
func (dim *DeviceInfoManager) getCapabilities(info *DeviceInfo) error {
	soap := dim.buildGetCapabilitiesSOAP(info)

	var resp string
	var err error

	if info.AuthType == "digest" {
		resp, err = dim.sendSOAPRequestWithDigest(info.XAddr, soap, info.Username, info.Password)
	} else {
		resp, err = dim.sendSOAPRequest(info.XAddr, soap)
	}

	if err != nil {
		return err
	}

	// 使用 XML 解析
	type GetCapabilitiesResponse struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Response struct {
				Capabilities struct {
					Device struct {
						XAddr string `xml:"XAddr"`
					} `xml:"Device"`
					Media struct {
						XAddr string `xml:"XAddr"`
					} `xml:"Media"`
					Events struct {
						XAddr string `xml:"XAddr"`
					} `xml:"Events"`
					PTZ struct {
						XAddr string `xml:"XAddr"`
					} `xml:"PTZ"`
					Imaging struct {
						XAddr string `xml:"XAddr"`
					} `xml:"Imaging"`
					Extension struct {
						Media2 struct {
							XAddr string `xml:"XAddr"`
						} `xml:"Media2"`
					} `xml:"Extension"`
				} `xml:"Capabilities"`
			} `xml:"GetCapabilitiesResponse"`
		} `xml:"Body"`
	}

	var capResp GetCapabilitiesResponse
	if err := xml.Unmarshal([]byte(resp), &capResp); err != nil {
		return fmt.Errorf("解析设备能力失败: %v", err)
	}

	info.Capabilities.DeviceService = capResp.Body.Response.Capabilities.Device.XAddr
	info.Capabilities.MediaService = capResp.Body.Response.Capabilities.Media.XAddr
	info.Capabilities.EventService = capResp.Body.Response.Capabilities.Events.XAddr
	info.Capabilities.PTZService = capResp.Body.Response.Capabilities.PTZ.XAddr
	info.Capabilities.ImagingService = capResp.Body.Response.Capabilities.Imaging.XAddr

	info.MediaXAddr = info.Capabilities.MediaService
	if info.MediaXAddr == "" {
		// 某些设备可能使用 Media2
		info.MediaXAddr = capResp.Body.Response.Capabilities.Extension.Media2.XAddr
	}

	return nil
}

// getChannels 获取通道信息
func (dim *DeviceInfoManager) getChannels(info *DeviceInfo) error {
	if info.MediaXAddr == "" {
		return errors.New("媒体服务地址为空")
	}

	// 获取 Profiles
	soap := dim.buildGetProfilesSOAP(info)

	var resp string
	var err error

	if info.AuthType == "digest" {
		resp, err = dim.sendSOAPRequestWithDigest(info.MediaXAddr, soap, info.Username, info.Password)
	} else {
		resp, err = dim.sendSOAPRequest(info.MediaXAddr, soap)
	}

	if err != nil {
		return err
	}

	// 解析 Profiles
	profiles := dim.parseProfiles(resp)
	info.Channels = make([]ChannelInfo, 0)

	// 获取每个通道的流地址
	for i, profile := range profiles {
		channel := ChannelInfo{
			ChannelNo:   i + 1,
			Token:       profile.Token,
			ChannelName: profile.Name,
		}

		// 获取 RTSP 地址
		rtspUrl, err := dim.getStreamUri(info, profile.Token)
		if err == nil {
			channel.RTSPUrl = rtspUrl
			info.StreamProtocol = "RTSP"
		}

		info.Channels = append(info.Channels, channel)
	}

	info.TotalChannel = len(info.Channels)
	info.ChannelStart = 0 // ONVIF通常从0开始

	// 确定媒体传输协议
	info.MediaProtocol = "UDP/TCP" // RTSP支持两种

	return nil
}

// getStreamUri 获取流地址
func (dim *DeviceInfoManager) getStreamUri(info *DeviceInfo, profileToken string) (string, error) {
	soap := dim.buildGetStreamUriSOAP(info, profileToken)

	var resp string
	var err error

	if info.AuthType == "digest" {
		resp, err = dim.sendSOAPRequestWithDigest(info.MediaXAddr, soap, info.Username, info.Password)
	} else {
		resp, err = dim.sendSOAPRequest(info.MediaXAddr, soap)
	}

	if err != nil {
		return "", err
	}

	return extractTag(resp, "Uri"), nil
}

// Profile 媒体配置文件
type Profile struct {
	Token string
	Name  string
}

// parseProfiles 解析 GetProfiles 响应
func (dim *DeviceInfoManager) parseProfiles(resp string) []Profile {
	profiles := make([]Profile, 0)

	// 使用正则表达式提取所有 Profile
	tokenRe := regexp.MustCompile(`<.*?:Profiles.*?token="([^"]+)"`)
	tokenMatches := tokenRe.FindAllStringSubmatch(resp, -1)

	nameRe := regexp.MustCompile(`<.*?:Name>([^<]+)</.*?:Name>`)
	nameMatches := nameRe.FindAllStringSubmatch(resp, -1)

	// 组合 token 和 name
	for i, tokenMatch := range tokenMatches {
		profile := Profile{
			Token: tokenMatch[1],
		}

		// 尝试匹配对应的名称
		if i < len(nameMatches) {
			profile.Name = nameMatches[i][1]
		} else {
			profile.Name = fmt.Sprintf("Profile_%d", i+1)
		}

		profiles = append(profiles, profile)
	}

	return profiles
}

// extractTag 从 XML 响应中提取指定标签的内容
func extractTag(xml, tag string) string {
	// 尝试带命名空间的标签
	patterns := []string{
		fmt.Sprintf(`<.*?:%s>([^<]+)</.*?:%s>`, tag, tag),
		fmt.Sprintf(`<%s>([^<]+)</%s>`, tag, tag),
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(xml)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// SOAP 请求构建

func (dim *DeviceInfoManager) buildGetDeviceInformationSOAP(info *DeviceInfo, authType string) string {
	if authType == "digest" {
		return `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<tds:GetDeviceInformation/>
	</s:Body>
</s:Envelope>`
	}

	// WS-Security
	created := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	nonce := generateNonce()
	digest := createDigest(nonce, created, info.Password)
	nonceBase64 := base64.StdEncoding.EncodeToString([]byte(nonce))

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Header>
		<Security s:mustUnderstand="1" xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
			<UsernameToken>
				<Username>%s</Username>
				<Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
				<Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</Nonce>
				<Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">%s</Created>
			</UsernameToken>
		</Security>
	</s:Header>
	<s:Body>
		<tds:GetDeviceInformation/>
	</s:Body>
</s:Envelope>`, info.Username, digest, nonceBase64, created)
}

func (dim *DeviceInfoManager) buildGetCapabilitiesSOAP(info *DeviceInfo) string {
	if info.AuthType == "digest" {
		return `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<tds:GetCapabilities>
			<tds:Category>All</tds:Category>
		</tds:GetCapabilities>
	</s:Body>
</s:Envelope>`
	}

	created := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	nonce := generateNonce()
	digest := createDigest(nonce, created, info.Password)
	nonceBase64 := base64.StdEncoding.EncodeToString([]byte(nonce))

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Header>
		<Security s:mustUnderstand="1" xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
			<UsernameToken>
				<Username>%s</Username>
				<Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
				<Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</Nonce>
				<Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">%s</Created>
			</UsernameToken>
		</Security>
	</s:Header>
	<s:Body>
		<tds:GetCapabilities>
			<tds:Category>All</tds:Category>
		</tds:GetCapabilities>
	</s:Body>
</s:Envelope>`, info.Username, digest, nonceBase64, created)
}

func (dim *DeviceInfoManager) buildGetProfilesSOAP(info *DeviceInfo) string {
	if info.AuthType == "digest" {
		return `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:trt="http://www.onvif.org/ver10/media/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<trt:GetProfiles/>
	</s:Body>
</s:Envelope>`
	}

	created := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	nonce := generateNonce()
	digest := createDigest(nonce, created, info.Password)
	nonceBase64 := base64.StdEncoding.EncodeToString([]byte(nonce))

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:trt="http://www.onvif.org/ver10/media/wsdl">
	<s:Header>
		<Security s:mustUnderstand="1" xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
			<UsernameToken>
				<Username>%s</Username>
				<Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
				<Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</Nonce>
				<Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">%s</Created>
			</UsernameToken>
		</Security>
	</s:Header>
	<s:Body>
		<trt:GetProfiles/>
	</s:Body>
</s:Envelope>`, info.Username, digest, nonceBase64, created)
}

func (dim *DeviceInfoManager) buildGetStreamUriSOAP(info *DeviceInfo, profileToken string) string {
	if info.AuthType == "digest" {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:trt="http://www.onvif.org/ver10/media/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema">
	<s:Header></s:Header>
	<s:Body>
		<trt:GetStreamUri>
			<trt:StreamSetup>
				<tt:Stream>RTP-Unicast</tt:Stream>
				<tt:Transport>
					<tt:Protocol>RTSP</tt:Protocol>
				</tt:Transport>
			</trt:StreamSetup>
			<trt:ProfileToken>%s</trt:ProfileToken>
		</trt:GetStreamUri>
	</s:Body>
</s:Envelope>`, profileToken)
	}

	created := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	nonce := generateNonce()
	digest := createDigest(nonce, created, info.Password)
	nonceBase64 := base64.StdEncoding.EncodeToString([]byte(nonce))

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:trt="http://www.onvif.org/ver10/media/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema">
	<s:Header>
		<Security s:mustUnderstand="1" xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
			<UsernameToken>
				<Username>%s</Username>
				<Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
				<Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</Nonce>
				<Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">%s</Created>
			</UsernameToken>
		</Security>
	</s:Header>
	<s:Body>
		<trt:GetStreamUri>
			<trt:StreamSetup>
				<tt:Stream>RTP-Unicast</tt:Stream>
				<tt:Transport>
					<tt:Protocol>RTSP</tt:Protocol>
				</tt:Transport>
			</trt:StreamSetup>
			<trt:ProfileToken>%s</trt:ProfileToken>
		</trt:GetStreamUri>
	</s:Body>
</s:Envelope>`, info.Username, digest, nonceBase64, created, profileToken)
}

// HTTP 请求

func (dim *DeviceInfoManager) sendSOAPRequest(xaddr, soap string) (string, error) {
	req, err := http.NewRequest("POST", xaddr, strings.NewReader(soap))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("Accept", "application/soap+xml")

	resp, err := dim.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	bodyStr := string(body)

	// 检查状态码
	if resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("401 Unauthorized")
	}

	// 检查是否是 SOAP Fault (认证失败)
	if strings.Contains(bodyStr, "NotAuthorized") ||
		strings.Contains(bodyStr, "Sender not Authorized") ||
		strings.Contains(bodyStr, ":Fault") {
		return "", errors.New("SOAP Fault: 认证失败")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP错误 %d", resp.StatusCode)
	}

	return bodyStr, nil
}

func (dim *DeviceInfoManager) sendSOAPRequestWithDigest(xaddr, soap, username, password string) (string, error) {
	// 第一次请求
	req, _ := http.NewRequest("POST", xaddr, strings.NewReader(soap))
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	resp, err := dim.client.Do(req)
	if err != nil {
		return "", err
	}

	// 如果不是401,读取响应
	if resp.StatusCode != http.StatusUnauthorized {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(body)

		// 检查是否是有效的 ONVIF 响应
		if resp.StatusCode == http.StatusOK && strings.Contains(bodyStr, "Envelope") {
			return bodyStr, nil
		}

		return "", fmt.Errorf("Digest认证失败: 状态码 %d", resp.StatusCode)
	}

	// 401 - 需要 Digest 认证
	authHeader := resp.Header.Get("WWW-Authenticate")
	resp.Body.Close()

	if authHeader == "" || !strings.HasPrefix(authHeader, "Digest") {
		return "", errors.New("设备不支持 Digest 认证")
	}

	authParams := parseDigestAuthHeader(authHeader)
	authResp := generateDigestAuth(username, password, "POST", xaddr, authParams)

	// 第二次请求,携带 Digest 认证
	req2, _ := http.NewRequest("POST", xaddr, strings.NewReader(soap))
	req2.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req2.Header.Set("Authorization", authResp)

	resp2, err := dim.client.Do(req2)
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()

	body, _ := ioutil.ReadAll(resp2.Body)
	bodyStr := string(body)

	// 检查认证是否成功
	if resp2.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Digest认证失败: 状态码 %d", resp2.StatusCode)
	}

	// 检查是否是 SOAP Fault
	if strings.Contains(bodyStr, "NotAuthorized") || strings.Contains(bodyStr, "Unauthorized") {
		return "", errors.New("Digest认证失败: NotAuthorized")
	}

	return bodyStr, nil
}

func (dim *DeviceInfoManager) parseBasicInfo(info *DeviceInfo) {
	xaddr := info.XAddr

	// 解析协议
	if strings.HasPrefix(xaddr, "https://") {
		info.Protocol = "https"
		xaddr = strings.TrimPrefix(xaddr, "https://")
	} else {
		info.Protocol = "http"
		xaddr = strings.TrimPrefix(xaddr, "http://")
	}

	// 解析 IP 和端口
	parts := strings.Split(xaddr, "/")
	if len(parts) > 0 {
		hostPort := parts[0]
		hostParts := strings.Split(hostPort, ":")

		info.IPAddress = hostParts[0]

		if len(hostParts) > 1 {
			fmt.Sscanf(hostParts[1], "%d", &info.Port)
		} else {
			if info.Protocol == "https" {
				info.Port = 443
			} else {
				info.Port = 80
			}
		}
	}
}

// 工具函数

func generateNonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func createDigest(nonce, created, password string) string {
	hash := sha1.New()
	hash.Write([]byte(nonce))
	hash.Write([]byte(created))
	hash.Write([]byte(password))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func parseDigestAuthHeader(header string) map[string]string {
	params := make(map[string]string)
	header = strings.TrimPrefix(header, "Digest ")

	re := regexp.MustCompile(`(\w+)="?([^",]+)"?`)
	matches := re.FindAllStringSubmatch(header, -1)

	for _, match := range matches {
		if len(match) == 3 {
			params[match[1]] = match[2]
		}
	}

	return params
}

func generateDigestAuth(username, password, method, uri string, params map[string]string) string {
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]
	opaque := params["opaque"]

	cnonce := fmt.Sprintf("%d", time.Now().UnixNano())
	nc := "00000001"

	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", username, realm, password))

	uriPath := uri
	if idx := strings.Index(uri, "://"); idx != -1 {
		remaining := uri[idx+3:]
		if idx2 := strings.Index(remaining, "/"); idx2 != -1 {
			uriPath = remaining[idx2:]
		} else {
			uriPath = "/"
		}
	}

	ha2 := md5Hash(fmt.Sprintf("%s:%s", method, uriPath))

	var response string
	if qop == "auth" || qop == "auth-int" {
		response = md5Hash(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
	} else {
		response = md5Hash(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	}

	authStr := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		username, realm, nonce, uriPath, response)

	if qop != "" {
		authStr += fmt.Sprintf(`, qop=%s, nc=%s, cnonce="%s"`, qop, nc, cnonce)
	}

	if opaque != "" {
		authStr += fmt.Sprintf(`, opaque="%s"`, opaque)
	}

	return authStr
}

func md5Hash(text string) string {
	hash := md5.New()
	hash.Write([]byte(text))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
