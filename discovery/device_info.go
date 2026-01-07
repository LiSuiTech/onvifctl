package discovery

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// authenticate 尝试认证 - 增强版
func (dim *DeviceInfoManager) authenticateEnhanced(info *DeviceInfo) (string, error) {
	//fmt.Printf("  [认证测试] 设备地址: %s\n", info.XAddr)

	// 1. 先尝试无认证访问
	//fmt.Printf("  [认证测试] 尝试无认证访问...\n")
	soap := dim.buildGetDeviceInformationSOAPNoAuth()
	resp, err := dim.sendSOAPRequest(info.XAddr, soap)
	if err == nil && strings.Contains(resp, "GetDeviceInformationResponse") {
		//fmt.Printf("  [认证测试] ✓ 设备允许无认证访问\n")
		return "none", nil
	}

	// 2. 尝试 WS-Security 认证
	//fmt.Printf("  [认证测试] 尝试 WS-Security 认证...\n")
	soap = dim.buildGetDeviceInformationSOAP(info, "wsse")
	resp, err = dim.sendSOAPRequest(info.XAddr, soap)
	if err == nil {
		if strings.Contains(resp, "GetDeviceInformationResponse") {
			//fmt.Printf("  [认证测试] ✓ WS-Security 认证成功\n")
			return "wsse", nil
		}
		//if strings.Contains(resp, "NotAuthorized") || strings.Contains(resp, "Unauthorized") {
		//	fmt.Printf("  [认证测试] ✗ WS-Security 认证失败: 用户名或密码错误\n")
		//} else {
		//	fmt.Printf("  [认证测试] ✗ WS-Security 响应异常\n")
		//}
		//} else {
		//	fmt.Printf("  [认证测试] ✗ WS-Security 请求失败: %v\n", err)
	}

	// 3. 尝试 Digest 认证
	//fmt.Printf("  [认证测试] 尝试 Digest 认证...\n")
	soap = dim.buildGetDeviceInformationSOAP(info, "digest")
	resp, err = dim.sendSOAPRequestWithDigest(info.XAddr, soap, info.Username, info.Password)
	if err == nil {
		if strings.Contains(resp, "GetDeviceInformationResponse") {
			//fmt.Printf("  [认证测试] ✓ Digest 认证成功\n")
			return "digest", nil
		}
		//	if strings.Contains(resp, "NotAuthorized") || strings.Contains(resp, "Unauthorized") {
		//		fmt.Printf("  [认证测试] ✗ Digest 认证失败: 用户名或密码错误\n")
		//	} else {
		//		fmt.Printf("  [认证测试] ✗ Digest 响应异常\n")
		//	}
		//} else {
		//	fmt.Printf("  [认证测试] ✗ Digest 请求失败: %v\n", err)
	}

	// 4. 尝试基本的 HTTP Basic 认证
	//fmt.Printf("  [认证测试] 尝试 HTTP Basic 认证...\n")
	soap = dim.buildGetDeviceInformationSOAPNoAuth()
	resp, err = dim.sendSOAPRequestWithBasicAuth(info.XAddr, soap, info.Username, info.Password)
	if err == nil && strings.Contains(resp, "GetDeviceInformationResponse") {
		//fmt.Printf("  [认证测试] ✓ HTTP Basic 认证成功\n")
		return "basic", nil
	}

	return "", errors.New("所有认证方式都失败(none/wsse/digest/basic)")
}

// buildGetDeviceInformationSOAPNoAuth 构建无认证的 SOAP 请求
func (dim *DeviceInfoManager) buildGetDeviceInformationSOAPNoAuth() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<tds:GetDeviceInformation/>
	</s:Body>
</s:Envelope>`
}

// sendSOAPRequestWithBasicAuth 使用 HTTP Basic 认证发送 SOAP 请求
func (dim *DeviceInfoManager) sendSOAPRequestWithBasicAuth(xaddr, soap, username, password string) (string, error) {
	req, err := http.NewRequest("POST", xaddr, strings.NewReader(soap))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("Accept", "application/soap+xml")

	// 添加 Basic 认证头
	auth := username + ":" + password
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", basicAuth)

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

	if resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("401 Unauthorized")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP错误 %d", resp.StatusCode)
	}

	return bodyStr, nil
}

// GetDeviceInfoEnhanced 增强版获取设备信息(带详细日志)
func (dim *DeviceInfoManager) GetDeviceInfoEnhanced(xaddr string, verbose bool) (*DeviceInfo, error) {
	if verbose {
		fmt.Printf("\n========================================\n")
		fmt.Printf("获取设备信息: %s\n", xaddr)
		fmt.Printf("========================================\n")
	}

	// 尝试所有凭据
	for i, cred := range dim.credentials {
		if verbose {
			fmt.Printf("\n[凭据 %d/%d] 用户名: %s, 密码: %s\n", i+1, len(dim.credentials), cred.Username, strings.Repeat("*", len(cred.Password)))
		}

		info, err := dim.getDeviceInfoWithCredentialEnhanced(xaddr, cred.Username, cred.Password, verbose)
		if err == nil {
			if verbose {
				fmt.Printf("\n✓ 认证成功! 使用凭据: %s/%s\n", cred.Username, strings.Repeat("*", len(cred.Password)))
				fmt.Printf("✓ 认证方式: %s\n", info.AuthType)
			}
			return info, nil
		}

		if verbose {
			fmt.Printf("✗ 认证失败: %v\n", err)
		}
	}

	return nil, fmt.Errorf("所有凭据都认证失败")
}

// getDeviceInfoWithCredentialEnhanced 使用指定凭据获取设备信息(增强版)
func (dim *DeviceInfoManager) getDeviceInfoWithCredentialEnhanced(xaddr, username, password string, verbose bool) (*DeviceInfo, error) {
	info := &DeviceInfo{
		XAddr:    xaddr,
		Username: username,
		Password: password,
	}

	// 解析基本信息
	dim.parseBasicInfo(info)

	// 1. 尝试认证
	authType, err := dim.authenticateEnhanced(info)
	if err != nil {
		return nil, fmt.Errorf("认证失败: %v", err)
	}
	info.AuthType = authType

	if verbose {
		fmt.Printf("\n  [信息获取] 开始获取设备详细信息...\n")
	}

	// 2. 获取设备信息
	err = dim.getDeviceInformationWithAuth(info, verbose)
	if err != nil {
		if verbose {
			fmt.Printf("  [信息获取] ⚠ 获取设备信息失败: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("  [信息获取] ✓ 厂商: %s, 型号: %s\n", info.Manufacturer, info.Model)
	}

	// 3. 获取设备能力
	err = dim.getCapabilitiesWithAuth(info, verbose)
	if err != nil {
		if verbose {
			fmt.Printf("  [信息获取] ⚠ 获取设备能力失败: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("  [信息获取] ✓ 媒体服务: %s\n", info.MediaXAddr)
	}

	// 4. 获取通道信息
	err = dim.getChannelsWithAuth(info, verbose)
	if err != nil {
		if verbose {
			fmt.Printf("  [信息获取] ⚠ 获取通道信息失败: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("  [信息获取] ✓ 通道数: %d\n", len(info.Channels))
	}

	return info, nil
}

// getDeviceInformationWithAuth 使用认证获取设备信息
func (dim *DeviceInfoManager) getDeviceInformationWithAuth(info *DeviceInfo, verbose bool) error {
	var soap string
	var resp string
	var err error

	switch info.AuthType {
	case "none":
		soap = dim.buildGetDeviceInformationSOAPNoAuth()
		resp, err = dim.sendSOAPRequest(info.XAddr, soap)
	case "basic":
		soap = dim.buildGetDeviceInformationSOAPNoAuth()
		resp, err = dim.sendSOAPRequestWithBasicAuth(info.XAddr, soap, info.Username, info.Password)
	case "digest":
		soap = dim.buildGetDeviceInformationSOAP(info, "digest")
		resp, err = dim.sendSOAPRequestWithDigest(info.XAddr, soap, info.Username, info.Password)
	default: // wsse
		soap = dim.buildGetDeviceInformationSOAP(info, "wsse")
		resp, err = dim.sendSOAPRequest(info.XAddr, soap)
	}

	if err != nil {
		return err
	}

	// 解析 XML
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

// getCapabilitiesWithAuth 使用认证获取设备能力
func (dim *DeviceInfoManager) getCapabilitiesWithAuth(info *DeviceInfo, verbose bool) error {
	var soap string
	var resp string
	var err error

	switch info.AuthType {
	case "none":
		soap = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<tds:GetCapabilities>
			<tds:Category>All</tds:Category>
		</tds:GetCapabilities>
	</s:Body>
</s:Envelope>`
		resp, err = dim.sendSOAPRequest(info.XAddr, soap)
	case "basic":
		soap = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<tds:GetCapabilities>
			<tds:Category>All</tds:Category>
		</tds:GetCapabilities>
	</s:Body>
</s:Envelope>`
		resp, err = dim.sendSOAPRequestWithBasicAuth(info.XAddr, soap, info.Username, info.Password)
	case "digest":
		soap = dim.buildGetCapabilitiesSOAP(info)
		resp, err = dim.sendSOAPRequestWithDigest(info.XAddr, soap, info.Username, info.Password)
	default: // wsse
		soap = dim.buildGetCapabilitiesSOAP(info)
		resp, err = dim.sendSOAPRequest(info.XAddr, soap)
	}

	if err != nil {
		return err
	}

	// 解析 XML
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

	return nil
}

// getChannelsWithAuth 使用认证获取通道信息
func (dim *DeviceInfoManager) getChannelsWithAuth(info *DeviceInfo, verbose bool) error {
	if info.MediaXAddr == "" {
		return errors.New("媒体服务地址为空")
	}

	// 获取 Profiles
	var soap string
	var resp string
	var err error

	switch info.AuthType {
	case "none":
		soap = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:trt="http://www.onvif.org/ver10/media/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<trt:GetProfiles/>
	</s:Body>
</s:Envelope>`
		resp, err = dim.sendSOAPRequest(info.MediaXAddr, soap)
	case "basic":
		soap = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:trt="http://www.onvif.org/ver10/media/wsdl">
	<s:Header></s:Header>
	<s:Body>
		<trt:GetProfiles/>
	</s:Body>
</s:Envelope>`
		resp, err = dim.sendSOAPRequestWithBasicAuth(info.MediaXAddr, soap, info.Username, info.Password)
	case "digest":
		soap = dim.buildGetProfilesSOAP(info)
		resp, err = dim.sendSOAPRequestWithDigest(info.MediaXAddr, soap, info.Username, info.Password)
	default: // wsse
		soap = dim.buildGetProfilesSOAP(info)
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
		rtspUrl, err := dim.getStreamUriWithAuth(info, profile.Token)
		if err == nil {
			channel.RTSPUrl = rtspUrl
			info.StreamProtocol = "RTSP"
		}

		info.Channels = append(info.Channels, channel)
	}

	info.TotalChannel = len(info.Channels)
	info.ChannelStart = 0

	info.MediaProtocol = "UDP/TCP"

	return nil
}

// getStreamUriWithAuth 使用认证获取流地址
func (dim *DeviceInfoManager) getStreamUriWithAuth(info *DeviceInfo, profileToken string) (string, error) {
	var soap string
	var resp string
	var err error

	switch info.AuthType {
	case "none":
		soap = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
		resp, err = dim.sendSOAPRequest(info.MediaXAddr, soap)
	case "basic":
		soap = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
		resp, err = dim.sendSOAPRequestWithBasicAuth(info.MediaXAddr, soap, info.Username, info.Password)
	case "digest":
		soap = dim.buildGetStreamUriSOAP(info, profileToken)
		resp, err = dim.sendSOAPRequestWithDigest(info.MediaXAddr, soap, info.Username, info.Password)
	default: // wsse
		soap = dim.buildGetStreamUriSOAP(info, profileToken)
		resp, err = dim.sendSOAPRequest(info.MediaXAddr, soap)
	}

	if err != nil {
		return "", err
	}

	return extractTag(resp, "Uri"), nil
}
