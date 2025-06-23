// pressure/types.go - 壓差儀相關數據類型和常量定義
package pressure

import (
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// 基本數據類型
// ============================================================================

// DataFormatType 數據格式類型
type DataFormatType int

const (
	DecimalFormat DataFormatType = 0 // 十進制格式 (擴大10倍)
	FloatFormat   DataFormatType = 1 // IEEE 754 浮點數格式
)

// String 實現 Stringer 接口
func (dft DataFormatType) String() string {
	switch dft {
	case DecimalFormat:
		return "decimal"
	case FloatFormat:
		return "float"
	default:
		return "unknown"
	}
}

// MarshalText 實現 encoding.TextMarshaler 接口，用於 JSON/YAML 序列化
func (dft DataFormatType) MarshalText() ([]byte, error) {
	return []byte(dft.String()), nil
}

// UnmarshalText 實現 encoding.TextUnmarshaler 接口，用於 JSON/YAML 反序列化
func (dft *DataFormatType) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "decimal", "dec", "0":
		*dft = DecimalFormat
	case "float", "floating", "1":
		*dft = FloatFormat
	default:
		return fmt.Errorf("unknown data format: %s", string(text))
	}
	return nil
}

// ============================================================================
// 設備狀態相關類型
// ============================================================================

// DeviceStatus 設備運行狀態
type DeviceStatus int

const (
	StatusStopped      DeviceStatus = 0 // 停止
	StatusRunning      DeviceStatus = 1 // 運行中
	StatusError        DeviceStatus = 2 // 錯誤狀態
	StatusConnecting   DeviceStatus = 3 // 連接中
	StatusDisconnected DeviceStatus = 4 // 斷開連接
)

// String 實現 Stringer 接口
func (ds DeviceStatus) String() string {
	switch ds {
	case StatusStopped:
		return "stopped"
	case StatusRunning:
		return "running"
	case StatusError:
		return "error"
	case StatusConnecting:
		return "connecting"
	case StatusDisconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}

// IsActive 檢查設備是否處於活躍狀態
func (ds DeviceStatus) IsActive() bool {
	return ds == StatusRunning || ds == StatusConnecting
}

// ============================================================================
// 壓力測量相關類型
// ============================================================================

// PressureUnit 壓力單位
type PressureUnit int

const (
	Pascal       PressureUnit = 0 // 帕斯卡 (Pa)
	Kilopascal   PressureUnit = 1 // 千帕 (kPa)
	Millibar     PressureUnit = 2 // 毫巴 (mbar)
	Torr         PressureUnit = 3 // 托 (Torr)
	PSI          PressureUnit = 4 // 磅力每平方英寸 (psi)
	InchH2O      PressureUnit = 5 // 英寸水柱 (inH2O)
	MmH2O        PressureUnit = 6 // 毫米水柱 (mmH2O)
	AtmTechnical PressureUnit = 7 // 工程大氣壓 (at)
)

// String 實現 Stringer 接口
func (pu PressureUnit) String() string {
	switch pu {
	case Pascal:
		return "Pa"
	case Kilopascal:
		return "kPa"
	case Millibar:
		return "mbar"
	case Torr:
		return "Torr"
	case PSI:
		return "psi"
	case InchH2O:
		return "inH2O"
	case MmH2O:
		return "mmH2O"
	case AtmTechnical:
		return "at"
	default:
		return "unknown"
	}
}

// Symbol 返回壓力單位符號
func (pu PressureUnit) Symbol() string {
	return pu.String()
}

// ConvertFromPascal 從帕斯卡轉換到指定單位
func (pu PressureUnit) ConvertFromPascal(pascalValue float64) float64 {
	switch pu {
	case Pascal:
		return pascalValue
	case Kilopascal:
		return pascalValue / 1000.0
	case Millibar:
		return pascalValue / 100.0
	case Torr:
		return pascalValue / 133.322
	case PSI:
		return pascalValue / 6894.757
	case InchH2O:
		return pascalValue / 249.089
	case MmH2O:
		return pascalValue / 9.80665
	case AtmTechnical:
		return pascalValue / 98066.5
	default:
		return pascalValue
	}
}

// ConvertToPascal 從指定單位轉換到帕斯卡
func (pu PressureUnit) ConvertToPascal(value float64) float64 {
	switch pu {
	case Pascal:
		return value
	case Kilopascal:
		return value * 1000.0
	case Millibar:
		return value * 100.0
	case Torr:
		return value * 133.322
	case PSI:
		return value * 6894.757
	case InchH2O:
		return value * 249.089
	case MmH2O:
		return value * 9.80665
	case AtmTechnical:
		return value * 98066.5
	default:
		return value
	}
}

// ============================================================================
// 測量數據類型
// ============================================================================

// Measurement 壓力測量值（帶單位）
type Measurement struct {
	Value float64      `json:"value"` // 數值
	Unit  PressureUnit `json:"unit"`  // 單位
}

// String 實現 Stringer 接口
func (m Measurement) String() string {
	return fmt.Sprintf("%.3f %s", m.Value, m.Unit.Symbol())
}

// ToPascal 轉換為帕斯卡
func (m Measurement) ToPascal() float64 {
	return m.Unit.ConvertToPascal(m.Value)
}

// To 轉換為指定單位
func (m Measurement) To(unit PressureUnit) Measurement {
	pascalValue := m.ToPascal()
	return Measurement{
		Value: unit.ConvertFromPascal(pascalValue),
		Unit:  unit,
	}
}

// ============================================================================
// 錯誤類型
// ============================================================================

// ErrorCode 錯誤代碼
type ErrorCode int

const (
	ErrNone           ErrorCode = 0  // 無錯誤
	ErrConnection     ErrorCode = 1  // 連接錯誤
	ErrTimeout        ErrorCode = 2  // 超時錯誤
	ErrInvalidData    ErrorCode = 3  // 無效數據
	ErrDeviceNotFound ErrorCode = 4  // 設備未找到
	ErrPermission     ErrorCode = 5  // 權限錯誤
	ErrConfig         ErrorCode = 6  // 配置錯誤
	ErrProtocol       ErrorCode = 7  // 協議錯誤
	ErrHardware       ErrorCode = 8  // 硬件錯誤
	ErrSoftware       ErrorCode = 9  // 軟件錯誤
	ErrUnknown        ErrorCode = 99 // 未知錯誤
)

// String 實現 Stringer 接口
func (ec ErrorCode) String() string {
	switch ec {
	case ErrNone:
		return "none"
	case ErrConnection:
		return "connection"
	case ErrTimeout:
		return "timeout"
	case ErrInvalidData:
		return "invalid_data"
	case ErrDeviceNotFound:
		return "device_not_found"
	case ErrPermission:
		return "permission"
	case ErrConfig:
		return "config"
	case ErrProtocol:
		return "protocol"
	case ErrHardware:
		return "hardware"
	case ErrSoftware:
		return "software"
	default:
		return "unknown"
	}
}

// Description 返回錯誤描述
func (ec ErrorCode) Description() string {
	switch ec {
	case ErrNone:
		return "無錯誤"
	case ErrConnection:
		return "連接錯誤"
	case ErrTimeout:
		return "操作超時"
	case ErrInvalidData:
		return "數據無效"
	case ErrDeviceNotFound:
		return "設備未找到"
	case ErrPermission:
		return "權限不足"
	case ErrConfig:
		return "配置錯誤"
	case ErrProtocol:
		return "協議錯誤"
	case ErrHardware:
		return "硬件故障"
	case ErrSoftware:
		return "軟件錯誤"
	default:
		return "未知錯誤"
	}
}

// PressureError 壓差儀專用錯誤類型
type PressureError struct {
	Code      ErrorCode `json:"code"`      // 錯誤代碼
	Message   string    `json:"message"`   // 錯誤消息
	Timestamp time.Time `json:"timestamp"` // 錯誤時間
	SlaveID   byte      `json:"slave_id"`  // 設備ID
	Context   string    `json:"context"`   // 錯誤上下文
}

// Error 實現 error 接口
func (pe PressureError) Error() string {
	return fmt.Sprintf("[%s] 站點%d: %s - %s",
		pe.Code, pe.SlaveID, pe.Message, pe.Context)
}

// NewPressureError 創建新的壓差儀錯誤
func NewPressureError(code ErrorCode, message string, slaveID byte) *PressureError {
	return &PressureError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		SlaveID:   slaveID,
	}
}

// WithContext 添加錯誤上下文
func (pe *PressureError) WithContext(context string) *PressureError {
	pe.Context = context
	return pe
}

// ============================================================================
// 統計類型
// ============================================================================

// Statistics 壓力統計信息
type Statistics struct {
	Count    int       `json:"count"`     // 樣本數量
	Min      float64   `json:"min"`       // 最小值
	Max      float64   `json:"max"`       // 最大值
	Mean     float64   `json:"mean"`      // 平均值
	StdDev   float64   `json:"std_dev"`   // 標準偏差
	LastTime time.Time `json:"last_time"` // 最後更新時間
}

// Update 更新統計信息
func (s *Statistics) Update(value float64) {
	if s.Count == 0 {
		s.Min = value
		s.Max = value
		s.Mean = value
	} else {
		if value < s.Min {
			s.Min = value
		}
		if value > s.Max {
			s.Max = value
		}

		// 增量計算平均值
		oldMean := s.Mean
		s.Mean = oldMean + (value-oldMean)/float64(s.Count+1)

		// 增量計算標準偏差（Welford's algorithm）
		if s.Count > 0 {
			s.StdDev = s.StdDev + (value-oldMean)*(value-s.Mean)
		}
	}

	s.Count++
	s.LastTime = time.Now()

	// 計算最終標準偏差
	if s.Count > 1 {
		s.StdDev = s.StdDev / float64(s.Count-1)
	}
}

// Reset 重置統計信息
func (s *Statistics) Reset() {
	*s = Statistics{}
}

// String 實現 Stringer 接口
func (s Statistics) String() string {
	if s.Count == 0 {
		return "統計: 無數據"
	}
	return fmt.Sprintf("統計: 數量=%d, 範圍=[%.2f, %.2f], 平均=%.2f, 標準偏差=%.2f",
		s.Count, s.Min, s.Max, s.Mean, s.StdDev)
}

// ============================================================================
// 設備信息類型
// ============================================================================

// DeviceModel 設備型號信息
type DeviceModel struct {
	Manufacturer string `json:"manufacturer"` // 製造商
	Model        string `json:"model"`        // 型號
	Version      string `json:"version"`      // 版本
	Description  string `json:"description"`  // 描述
}

// String 實現 Stringer 接口
func (dm DeviceModel) String() string {
	return fmt.Sprintf("%s %s v%s", dm.Manufacturer, dm.Model, dm.Version)
}

// FullName 返回完整名稱
func (dm DeviceModel) FullName() string {
	return fmt.Sprintf("%s %s", dm.Manufacturer, dm.Model)
}

// ============================================================================
// 配置驗證類型
// ============================================================================

// ValidationLevel 驗證級別
type ValidationLevel int

const (
	ValidationNone   ValidationLevel = 0 // 不驗證
	ValidationBasic  ValidationLevel = 1 // 基本驗證
	ValidationStrict ValidationLevel = 2 // 嚴格驗證
)

// String 實現 Stringer 接口
func (vl ValidationLevel) String() string {
	switch vl {
	case ValidationNone:
		return "none"
	case ValidationBasic:
		return "basic"
	case ValidationStrict:
		return "strict"
	default:
		return "unknown"
	}
}

// ============================================================================
// 事件類型
// ============================================================================

// EventType 事件類型
type EventType int

const (
	EventDeviceConnected    EventType = 1  // 設備連接
	EventDeviceDisconnected EventType = 2  // 設備斷開
	EventReadingReceived    EventType = 3  // 接收到讀數
	EventReadingError       EventType = 4  // 讀數錯誤
	EventConfigChanged      EventType = 5  // 配置更改
	EventScanStarted        EventType = 6  // 掃描開始
	EventScanCompleted      EventType = 7  // 掃描完成
	EventDeviceFound        EventType = 8  // 發現設備
	EventStatusChanged      EventType = 9  // 狀態更改
	EventAlarmTriggered     EventType = 10 // 告警觸發
)

// String 實現 Stringer 接口
func (et EventType) String() string {
	switch et {
	case EventDeviceConnected:
		return "device_connected"
	case EventDeviceDisconnected:
		return "device_disconnected"
	case EventReadingReceived:
		return "reading_received"
	case EventReadingError:
		return "reading_error"
	case EventConfigChanged:
		return "config_changed"
	case EventScanStarted:
		return "scan_started"
	case EventScanCompleted:
		return "scan_completed"
	case EventDeviceFound:
		return "device_found"
	case EventStatusChanged:
		return "status_changed"
	case EventAlarmTriggered:
		return "alarm_triggered"
	default:
		return "unknown"
	}
}

// Description 返回事件描述
func (et EventType) Description() string {
	switch et {
	case EventDeviceConnected:
		return "設備已連接"
	case EventDeviceDisconnected:
		return "設備已斷開"
	case EventReadingReceived:
		return "接收到壓力讀數"
	case EventReadingError:
		return "壓力讀數錯誤"
	case EventConfigChanged:
		return "配置已更改"
	case EventScanStarted:
		return "設備掃描開始"
	case EventScanCompleted:
		return "設備掃描完成"
	case EventDeviceFound:
		return "發現新設備"
	case EventStatusChanged:
		return "設備狀態更改"
	case EventAlarmTriggered:
		return "告警觸發"
	default:
		return "未知事件"
	}
}

// ============================================================================
// 常量定義
// ============================================================================

const (
	// Modbus 協議常量
	ModbusFunctionReadHoldingRegisters = 0x03
	ModbusMaxSlaveID                   = 247
	ModbusMinSlaveID                   = 1

	// 普時達壓差儀特定常量
	PushidaPressureRegisterAddr  = 0x0034 // 壓力寄存器地址
	PushidaPressureRegisterCount = 0x0002 // 壓力寄存器數量

	// 默認配置值
	DefaultBaudRate     = 9600
	DefaultTimeout      = 5 * time.Second
	DefaultReadInterval = 1 * time.Second
	DefaultSlaveID      = 0x16 // 22

	// 壓力範圍常量 (Pa)
	MinReasonablePressure = -50000.0 // 最小合理壓力值
	MaxReasonablePressure = 50000.0  // 最大合理壓力值

	// 緩衝區大小
	DefaultReadingBufferSize = 100
	DefaultEventBufferSize   = 50

	// 版本信息
	LibraryVersion = "1.0.0"
	LibraryName    = "pressure-meter-macArm64"
)

// ============================================================================
// 初始化函數
// ============================================================================

// GetSupportedBaudRates 獲取支援的波特率列表
func GetSupportedBaudRates() []int {
	return []int{1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200}
}

// GetCommonSlaveIDs 獲取常用的從站ID列表
func GetCommonSlaveIDs() []byte {
	return []byte{0x01, 0x02, 0x03, 0x16, 0x17, 0x18} // 1, 2, 3, 22, 23, 24
}

// IsValidSlaveID 檢查從站ID是否有效
func IsValidSlaveID(slaveID byte) bool {
	return slaveID >= ModbusMinSlaveID && slaveID <= ModbusMaxSlaveID
}

// IsValidBaudRate 檢查波特率是否支援
func IsValidBaudRate(baudRate int) bool {
	supported := GetSupportedBaudRates()
	for _, rate := range supported {
		if rate == baudRate {
			return true
		}
	}
	return false
}

// IsReasonablePressure 檢查壓力值是否在合理範圍內
func IsReasonablePressure(pressure float64) bool {
	return pressure >= MinReasonablePressure && pressure <= MaxReasonablePressure
}
