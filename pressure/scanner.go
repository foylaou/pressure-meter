// pressure/scanner.go - 壓差儀設備自動掃描和發現
package pressure

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/goburrow/modbus"
	"go.bug.st/serial"
)

// DeviceInfo 設備信息
type DeviceInfo struct {
	Device      string                 `json:"device"`       // 串口設備路徑
	SlaveID     byte                   `json:"slave_id"`     // 站點號
	Responsive  bool                   `json:"responsive"`   // 是否響應
	DataFormat  DataFormatType         `json:"data_format"`  // 數據格式
	LastReading *PressureReading       `json:"last_reading"` // 最後讀數
	Properties  map[string]interface{} `json:"properties"`   // 其他屬性
	ScanTime    time.Time              `json:"scan_time"`    // 掃描時間
	Error       string                 `json:"error"`        // 錯誤信息
}

// Scanner 設備掃描器
type Scanner struct {
	logger        *log.Logger
	scanTimeout   time.Duration
	deviceTimeout time.Duration
	verbose       bool
}

// ScanConfig 掃描配置
type ScanConfig struct {
	// SerialPorts 要掃描的串口列表，為空則自動檢測
	SerialPorts []string `json:"serial_ports"`
	// SlaveIDs 要掃描的從站ID範圍
	SlaveIDs []byte `json:"slave_ids"`
	// BaudRates 要嘗試的波特率
	BaudRates []int `json:"baud_rates"`
	// ScanTimeout 每個設備的掃描超時時間
	ScanTimeout time.Duration `json:"scan_timeout"`
	// MaxDevices 最大掃描設備數量
	MaxDevices int `json:"max_devices"`
	// AutoDetectFormat 是否自動檢測數據格式
	AutoDetectFormat bool `json:"auto_detect_format"`
	// Parallel 是否並行掃描
	Parallel bool `json:"parallel"`
	// SkipUnresponsive 是否跳過無響應的設備
	SkipUnresponsive bool `json:"skip_unresponsive"`
}

// ScanResult 掃描結果
type ScanResult struct {
	Devices     []DeviceInfo  `json:"devices"`      // 發現的設備
	ScanTime    time.Duration `json:"scan_time"`    // 掃描總時間
	TotalTested int           `json:"total_tested"` // 測試的設備總數
	Successful  int           `json:"successful"`   // 成功響應的設備數
	Config      ScanConfig    `json:"config"`       // 使用的掃描配置
}

// NewScanner 創建新的掃描器
func NewScanner(logger *log.Logger) *Scanner {
	if logger == nil {
		logger = log.Default()
	}

	return &Scanner{
		logger:        logger,
		scanTimeout:   2 * time.Second,
		deviceTimeout: 500 * time.Millisecond,
		verbose:       true,
	}
}

// SetVerbose 設置詳細輸出
func (s *Scanner) SetVerbose(verbose bool) *Scanner {
	s.verbose = verbose
	return s
}

// SetTimeout 設置超時時間
func (s *Scanner) SetTimeout(scanTimeout, deviceTimeout time.Duration) *Scanner {
	s.scanTimeout = scanTimeout
	s.deviceTimeout = deviceTimeout
	return s
}

// GetDefaultScanConfig 獲取默認掃描配置
func GetDefaultScanConfig() ScanConfig {
	return ScanConfig{
		SerialPorts:      []string{},                        // 自動檢測
		SlaveIDs:         generateSlaveIDRange(1, 247),      // 全範圍掃描
		BaudRates:        []int{9600, 19200, 38400, 115200}, // 常用波特率
		ScanTimeout:      2 * time.Second,
		MaxDevices:       20,
		AutoDetectFormat: true,
		Parallel:         false, // 默認串行掃描，避免串口衝突
		SkipUnresponsive: true,
	}
}

// GetQuickScanConfig 獲取快速掃描配置
func GetQuickScanConfig() ScanConfig {
	return ScanConfig{
		SerialPorts:      []string{},                                 // 自動檢測
		SlaveIDs:         []byte{0x16, 0x01, 0x02, 0x03, 0x04, 0x05}, // 常用站點號
		BaudRates:        []int{9600},                                // 只嘗試標準波特率
		ScanTimeout:      1 * time.Second,
		MaxDevices:       10,
		AutoDetectFormat: true,
		Parallel:         false,
		SkipUnresponsive: true,
	}
}

// ScanDevices 掃描壓差儀設備
func (s *Scanner) ScanDevices(config ScanConfig) (*ScanResult, error) {
	startTime := time.Now()
	s.logf("🔍 開始掃描壓差儀設備...")

	result := &ScanResult{
		Devices: []DeviceInfo{},
		Config:  config,
	}

	serialPorts := config.SerialPorts

	// 如果沒有指定串口，自動檢測
	if len(serialPorts) == 0 {
		ports, err := s.detectSerialPorts()
		if err != nil {
			return nil, fmt.Errorf("自動檢測串口失敗: %v", err)
		}
		serialPorts = ports
	}

	s.logf("📍 發現 %d 個串口設備: %v", len(serialPorts), serialPorts)

	// 掃描每個串口
	for _, port := range serialPorts {
		s.logf("🔌 掃描串口: %s", port)

		portDevices := s.scanPort(port, config)
		for _, device := range portDevices {
			if !config.SkipUnresponsive || device.Responsive {
				result.Devices = append(result.Devices, device)
			}
			result.TotalTested++
			if device.Responsive {
				result.Successful++
			}
		}

		if len(result.Devices) >= config.MaxDevices {
			s.logf("📊 已達到最大設備數量限制: %d", config.MaxDevices)
			break
		}
	}

	result.ScanTime = time.Since(startTime)
	s.logf("✅ 掃描完成，耗時 %v，發現 %d 個響應設備，測試了 %d 個配置",
		result.ScanTime, result.Successful, result.TotalTested)

	return result, nil
}

// detectSerialPorts 自動檢測系統中的串口設備
func (s *Scanner) detectSerialPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}

	var validPorts []string
	for _, port := range ports {
		// 過濾掉一些明顯不是 RS485 設備的串口
		if s.isLikelyRS485Port(port) {
			validPorts = append(validPorts, port)
		}
	}

	if len(validPorts) == 0 {
		s.logf("⚠️  未發現可能的 RS485 串口設備")
		// 如果沒有找到，返回所有串口讓用戶決定
		return ports, nil
	}

	return validPorts, nil
}

// isLikelyRS485Port 判斷串口是否可能是 RS485 設備
func (s *Scanner) isLikelyRS485Port(port string) bool {
	// 常見的 RS485 適配器模式
	patterns := []string{
		"ttyUSB", "ttyACM", "ttyS", // Linux
		"COM",                             // Windows
		"cu.usbserial", "cu.wchusbserial", // macOS
		"cu.SLAB_USBtoUART", // Silicon Labs CP210x
		"cu.usbmodem",       // USB CDC
	}

	portLower := strings.ToLower(port)
	for _, pattern := range patterns {
		if strings.Contains(portLower, strings.ToLower(pattern)) {
			return true
		}
	}

	// 排除一些明顯的系統設備
	excludePatterns := []string{
		"bluetooth", "irda", "printer",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(portLower, pattern) {
			return false
		}
	}

	return false
}

// scanPort 掃描指定串口上的設備
func (s *Scanner) scanPort(port string, config ScanConfig) []DeviceInfo {
	var devices []DeviceInfo

	// 嘗試不同的波特率
	for _, baudRate := range config.BaudRates {
		if s.verbose {
			s.logf("  📡 嘗試波特率: %d", baudRate)
		}

		portDevices := s.scanPortWithBaudRate(port, baudRate, config)
		if len(portDevices) > 0 {
			devices = append(devices, portDevices...)
			// 找到設備後通常不需要繼續嘗試其他波特率
			if s.hasResponsiveDevice(portDevices) {
				s.logf("  ✅ 在波特率 %d 找到響應設備，跳過其他波特率", baudRate)
				break
			}
		}
	}

	return devices
}

// hasResponsiveDevice 檢查設備列表中是否有響應的設備
func (s *Scanner) hasResponsiveDevice(devices []DeviceInfo) bool {
	for _, device := range devices {
		if device.Responsive {
			return true
		}
	}
	return false
}

// scanPortWithBaudRate 使用指定波特率掃描串口
func (s *Scanner) scanPortWithBaudRate(port string, baudRate int, config ScanConfig) []DeviceInfo {
	var devices []DeviceInfo

	// 掃描每個從站ID
	for _, slaveID := range config.SlaveIDs {
		device := s.testDevice(port, baudRate, slaveID, config)
		devices = append(devices, device)

		if device.Responsive && s.verbose {
			s.logf("    🎯 發現設備: 站點=%d, 壓力=%.1f Pa",
				slaveID, device.LastReading.Pressure)
		}

		if len(devices) >= config.MaxDevices {
			break
		}
	}

	return devices
}

// testDevice 測試特定設備是否響應
func (s *Scanner) testDevice(port string, baudRate int, slaveID byte, config ScanConfig) DeviceInfo {
	device := DeviceInfo{
		Device:     port,
		SlaveID:    slaveID,
		Responsive: false,
		Properties: make(map[string]interface{}),
		ScanTime:   time.Now(),
	}

	// 創建臨時 Modbus 連接
	handler := modbus.NewRTUClientHandler(port)
	handler.BaudRate = baudRate
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = slaveID
	handler.Timeout = config.ScanTimeout

	err := handler.Connect()
	if err != nil {
		device.Error = fmt.Sprintf("連接失敗: %v", err)
		return device
	}
	defer handler.Close()

	client := modbus.NewClient(handler)

	// 嘗試讀取壓力數據
	results, err := client.ReadHoldingRegisters(PressureRegisterAddr, RegisterCount)
	if err != nil {
		device.Error = fmt.Sprintf("讀取失敗: %v", err)
		return device
	}

	if len(results) == 4 {
		device.Responsive = true
		device.Properties["baud_rate"] = baudRate
		device.Properties["response_time"] = time.Since(device.ScanTime)

		// 如果啟用了自動檢測數據格式
		if config.AutoDetectFormat {
			dataFormat, confidence := s.detectDataFormat(results)
			device.DataFormat = dataFormat
			device.Properties["auto_detected_format"] = true
			device.Properties["format_confidence"] = confidence

			// 創建臨時讀數
			reading := PressureReading{
				Timestamp: time.Now(),
				SlaveID:   slaveID,
				RawData:   results,
				Valid:     true,
			}

			// 解析壓力值
			switch dataFormat {
			case DecimalFormat:
				reading.Pressure = parseDecimalFormatStatic(results)
			case FloatFormat:
				reading.Pressure = parseFloatFormatStatic(results)
			}

			device.LastReading = &reading
			device.Properties["pressure_pa"] = reading.Pressure
		}

		// 添加一些診斷信息
		device.Properties["raw_data"] = fmt.Sprintf("%02X %02X %02X %02X",
			results[0], results[1], results[2], results[3])
	}

	return device
}

// detectDataFormat 自動檢測數據格式，返回格式和置信度
func (s *Scanner) detectDataFormat(data []byte) (DataFormatType, float64) {
	// 嘗試解析為十進制格式
	decimalValue := parseDecimalFormatStatic(data)

	// 嘗試解析為浮點格式
	floatValue := parseFloatFormatStatic(data)

	// 計算置信度的啟發式規則
	decimalConfidence := s.calculateDecimalConfidence(decimalValue, data)
	floatConfidence := s.calculateFloatConfidence(floatValue, data)

	s.logf("      📊 格式檢測: 十進制=%.1f(置信度%.2f), 浮點=%.1f(置信度%.2f)",
		decimalValue, decimalConfidence, floatValue, floatConfidence)

	if decimalConfidence > floatConfidence {
		return DecimalFormat, decimalConfidence
	}
	return FloatFormat, floatConfidence
}

// calculateDecimalConfidence 計算十進制格式的置信度
func (s *Scanner) calculateDecimalConfidence(value float64, data []byte) float64 {
	confidence := 0.0

	// 如果值在合理的壓力範圍內 (-10000 到 10000 Pa)
	if value >= -10000 && value <= 10000 {
		confidence += 0.5
	}

	// 如果值是整數或一位小數（十進制格式特點）
	if value == float64(int(value*10))/10 {
		confidence += 0.3
	}

	// 如果原始數據看起來像十進制編碼
	if data[0] != 0xFF && (data[0] < 0x80 || data[0] == 0xFF) {
		confidence += 0.2
	}

	return confidence
}

// calculateFloatConfidence 計算浮點格式的置信度
func (s *Scanner) calculateFloatConfidence(value float64, data []byte) float64 {
	confidence := 0.0

	// 如果值在合理範圍內
	if value >= -10000 && value <= 10000 && !math.IsNaN(value) && !math.IsInf(value, 0) {
		confidence += 0.4
	}

	// 如果值有多位小數（浮點格式特點）
	if value != float64(int(value*10))/10 {
		confidence += 0.3
	}

	// 檢查 IEEE 754 格式的合理性
	ieeeBytes := make([]byte, 4)
	ieeeBytes[0] = data[2]
	ieeeBytes[1] = data[3]
	ieeeBytes[2] = data[0]
	ieeeBytes[3] = data[1]

	bits := binary.BigEndian.Uint32(ieeeBytes)
	exponent := (bits >> 23) & 0xFF

	// 正常的指數範圍
	if exponent > 0 && exponent < 255 {
		confidence += 0.3
	}

	return confidence
}

// AutoConfigure 自動配置第一個找到的設備
func (s *Scanner) AutoConfigure() (*Config, error) {
	s.logf("🚀 開始自動配置...")

	scanConfig := GetQuickScanConfig() // 使用快速掃描
	scanConfig.MaxDevices = 1          // 只需要找到一個設備

	result, err := s.ScanDevices(scanConfig)
	if err != nil {
		return nil, fmt.Errorf("掃描設備失敗: %v", err)
	}

	responsiveDevices := s.getResponsiveDevices(result.Devices)
	if len(responsiveDevices) == 0 {
		return nil, fmt.Errorf("未找到任何響應的壓差儀設備")
	}

	// 使用第一個找到的設備
	device := responsiveDevices[0]
	config := &Config{
		Device:       device.Device,
		SlaveID:      device.SlaveID,
		ReadInterval: time.Second,
		DataFormat:   device.DataFormat,
		Logger:       s.logger,
	}

	s.logf("✅ 自動配置完成: 設備=%s, 站點=%d, 格式=%v",
		config.Device, config.SlaveID, config.DataFormat)

	return config, nil
}

// QuickScan 快速掃描（僅掃描常用設備和參數）
func (s *Scanner) QuickScan() (*ScanResult, error) {
	s.logf("⚡ 開始快速掃描...")
	return s.ScanDevices(GetQuickScanConfig())
}

// FullScan 完整掃描
func (s *Scanner) FullScan() (*ScanResult, error) {
	s.logf("🔍 開始完整掃描...")
	return s.ScanDevices(GetDefaultScanConfig())
}

// getResponsiveDevices 獲取響應的設備列表
func (s *Scanner) getResponsiveDevices(devices []DeviceInfo) []DeviceInfo {
	var responsive []DeviceInfo
	for _, device := range devices {
		if device.Responsive {
			responsive = append(responsive, device)
		}
	}
	return responsive
}

// PrintScanResults 打印掃描結果
func (s *Scanner) PrintScanResults(result *ScanResult) {
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("📊 掃描結果 (耗時: %v)\n", result.ScanTime)
	fmt.Printf("🎯 測試了 %d 個配置，發現 %d 個響應設備\n", result.TotalTested, result.Successful)
	fmt.Println("=" + strings.Repeat("=", 50))

	responsiveDevices := s.getResponsiveDevices(result.Devices)

	if len(responsiveDevices) == 0 {
		fmt.Println("❌ 未找到任何響應的設備")
		fmt.Println("\n💡 建議:")
		fmt.Println("   - 檢查設備是否正確連接")
		fmt.Println("   - 確認設備電源是否開啟")
		fmt.Println("   - 檢查 RS485 接線是否正確")
		fmt.Println("   - 嘗試不同的波特率或站點號")
		return
	}

	for i, device := range responsiveDevices {
		fmt.Printf("\n🔌 設備 %d:\n", i+1)
		fmt.Printf("   串口: %s\n", device.Device)
		fmt.Printf("   站點號: %d (0x%02X)\n", device.SlaveID, device.SlaveID)

		if baudRate, ok := device.Properties["baud_rate"]; ok {
			fmt.Printf("   波特率: %v\n", baudRate)
		}

		fmt.Printf("   數據格式: %s", formatToString(device.DataFormat))
		if confidence, ok := device.Properties["format_confidence"]; ok {
			fmt.Printf(" (置信度: %.2f)", confidence)
		}
		fmt.Println()

		if device.LastReading != nil {
			fmt.Printf("   當前壓力: %.2f Pa\n", device.LastReading.Pressure)
		}

		if rawData, ok := device.Properties["raw_data"]; ok {
			fmt.Printf("   原始數據: %v\n", rawData)
		}

		if responseTime, ok := device.Properties["response_time"]; ok {
			fmt.Printf("   響應時間: %v\n", responseTime)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 52))
}

// logf 帶條件的日誌輸出
func (s *Scanner) logf(format string, args ...interface{}) {
	if s.verbose {
		s.logger.Printf(format, args...)
	}
}

// 輔助函數

// generateSlaveIDRange 生成從站ID範圍
func generateSlaveIDRange(start, end int) []byte {
	var ids []byte
	for i := start; i <= end; i++ {
		ids = append(ids, byte(i))
	}
	return ids
}

// 靜態解析函數（不依賴 PressureMeter 實例）

// parseDecimalFormatStatic 靜態解析十進制格式
func parseDecimalFormatStatic(data []byte) float64 {
	value := int32(binary.BigEndian.Uint32(data))
	if data[0] == 0xFF || (uint32(value)&0x80000000) == 0x80000000 {
		return float64(value) / 10.0
	}
	return float64(value) / 10.0
}

// parseFloatFormatStatic 靜態解析浮點格式
func parseFloatFormatStatic(data []byte) float64 {
	ieeeBytes := make([]byte, 4)
	ieeeBytes[0] = data[2]
	ieeeBytes[1] = data[3]
	ieeeBytes[2] = data[0]
	ieeeBytes[3] = data[1]

	bits := binary.BigEndian.Uint32(ieeeBytes)
	pressure := math.Float32frombits(bits)

	// 檢查是否為有效的浮點數
	if math.IsNaN(float64(pressure)) || math.IsInf(float64(pressure), 0) {
		return 0
	}

	return float64(pressure)
}
