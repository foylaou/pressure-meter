// pressure/device.go - 普時達壓差儀設備驅動核心代碼
package pressure

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/goburrow/modbus"
)

// Config 普時達壓差儀配置
type Config struct {
	// Device RS485 設備路徑 (如 /dev/ttyUSB0 或 COM1)
	Device string `json:"device" yaml:"device"`
	// SlaveID 儀表站點號 (1-247)
	SlaveID byte `json:"slaveid" yaml:"slaveid"`
	// ReadInterval 讀取間隔時間
	ReadInterval time.Duration `json:"readinterval" yaml:"readinterval"`
	// DataFormat 數據格式：0=十進制(默認), 1=浮點數
	DataFormat DataFormatType `json:"dataformat" yaml:"dataformat"`
	// Logger 日誌記錄器
	Logger *log.Logger `json:"-" yaml:"-"`
}

// PressureReading 壓力讀數
type PressureReading struct {
	Timestamp time.Time `json:"timestamp"` // 讀取時間
	Pressure  float64   `json:"pressure"`  // 壓力值 (Pa)
	SlaveID   byte      `json:"slave_id"`  // 設備 ID
	RawData   []byte    `json:"raw_data"`  // 原始數據
	Valid     bool      `json:"valid"`     // 數據是否有效
	Error     string    `json:"error"`     // 錯誤信息（如果有）
}

// PressureMeter 普時達壓差儀驅動
type PressureMeter struct {
	client     modbus.Client
	handler    *modbus.RTUClientHandler // 保存 handler 引用以便關閉連接
	slaveID    byte
	dataFormat DataFormatType
	logger     *log.Logger
	readings   chan PressureReading
	stopCh     chan struct{}
	running    bool
}

// Modbus 寄存器地址常量
const (
	PressureRegisterAddr = 0x0034 // 壓力數據寄存器地址
	RegisterCount        = 0x0002 // 讀取寄存器數量 (2個)
	FunctionCode         = 0x03   // 功能碼：讀保持寄存器
)

// NewPressureMeter 創建新的壓差儀實例
func NewPressureMeter(config Config) (*PressureMeter, error) {
	// 驗證配置
	if config.SlaveID < 1 || config.SlaveID > 247 {
		return nil, fmt.Errorf("invalid slave ID: %d, must be 1-247", config.SlaveID)
	}

	if config.ReadInterval == 0 {
		config.ReadInterval = time.Second // 默認 1 秒讀取一次
	}

	if config.Logger == nil {
		config.Logger = log.Default()
	}

	// 創建 Modbus RTU 客戶端處理器
	handler := modbus.NewRTUClientHandler(config.Device)
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = config.SlaveID
	handler.Timeout = 5 * time.Second

	// 連接設備
	err := handler.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to device %s: %v", config.Device, err)
	}

	// 創建 Modbus 客戶端
	client := modbus.NewClient(handler)

	pm := &PressureMeter{
		client:     client,
		handler:    handler, // 保存 handler 引用
		slaveID:    config.SlaveID,
		dataFormat: config.DataFormat,
		logger:     config.Logger,
		readings:   make(chan PressureReading, 100), // 緩衝 100 個讀數
		stopCh:     make(chan struct{}),
		running:    false,
	}

	return pm, nil
}

// Start 開始連續讀取壓力數據
func (pm *PressureMeter) Start(interval time.Duration) {
	if pm.running {
		pm.logger.Println("壓差儀已在運行中")
		return
	}

	pm.running = true
	pm.logger.Printf("開始讀取壓差儀數據，間隔: %v", interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-pm.stopCh:
				pm.logger.Println("停止讀取壓差儀數據")
				return
			case <-ticker.C:
				reading := pm.ReadPressure()
				select {
				case pm.readings <- reading:
				default:
					// 通道已滿，丟棄最舊的讀數
					pm.logger.Println("讀數通道已滿，丟棄舊數據")
					select {
					case <-pm.readings:
					default:
					}
					pm.readings <- reading
				}
			}
		}
	}()
}

// Stop 停止讀取
func (pm *PressureMeter) Stop() {
	if !pm.running {
		return
	}

	pm.running = false
	close(pm.stopCh)
	pm.logger.Println("已停止壓差儀讀取")
}

// ReadPressure 讀取一次壓力數據
func (pm *PressureMeter) ReadPressure() PressureReading {
	reading := PressureReading{
		Timestamp: time.Now(),
		SlaveID:   pm.slaveID,
		Valid:     false,
	}

	// 發送 Modbus 讀取命令
	// 功能碼 0x03, 地址 0x0034, 數量 0x0002
	results, err := pm.client.ReadHoldingRegisters(PressureRegisterAddr, RegisterCount)
	if err != nil {
		reading.Error = fmt.Sprintf("讀取壓力數據失敗: %v", err)
		pm.logger.Printf(reading.Error)
		return reading
	}

	if len(results) != 4 {
		reading.Error = fmt.Sprintf("接收數據長度錯誤: 期望4字節，實際%d字節", len(results))
		pm.logger.Printf(reading.Error)
		return reading
	}

	reading.RawData = make([]byte, len(results))
	copy(reading.RawData, results)

	// 根據數據格式解析壓力值
	switch pm.dataFormat {
	case DecimalFormat:
		reading.Pressure = pm.parseDecimalFormat(results)
	case FloatFormat:
		reading.Pressure = pm.parseFloatFormat(results)
	default:
		reading.Error = fmt.Sprintf("未知數據格式: %d", pm.dataFormat)
		pm.logger.Printf(reading.Error)
		return reading
	}

	reading.Valid = true
	pm.logger.Printf("讀取壓力: %.2f Pa (原始數據: %02X %02X %02X %02X)",
		reading.Pressure, results[0], results[1], results[2], results[3])

	return reading
}

// parseDecimalFormat 解析十進制格式數據
func (pm *PressureMeter) parseDecimalFormat(data []byte) float64 {
	// 組合 4 字節數據為 32 位整數
	// data[0] data[1] data[2] data[3] = D1 D2 D3 D4
	value := int32(binary.BigEndian.Uint32(data))

	// 檢查是否為負數
	// 方法1: 檢查最高字節是否為 0xFF
	if data[0] == 0xFF {
		pm.logger.Printf("檢測到負數 (最高字節 0xFF): %08X", uint32(value))
		// 對於負數，直接使用 int32 的值然後除以 10
		return float64(value) / 10.0
	}

	// 方法2: 檢查符號位
	if (uint32(value) & 0x80000000) == 0x80000000 {
		pm.logger.Printf("檢測到負數 (符號位): %08X", uint32(value))
		return float64(value) / 10.0
	}

	// 正數處理：除以 10 得到實際壓力值
	pressure := float64(value) / 10.0
	return pressure
}

// parseFloatFormat 解析浮點數格式數據 (IEEE 754, Modbus 3412 字節序)
func (pm *PressureMeter) parseFloatFormat(data []byte) float64 {
	// Modbus 3412 字節序轉換為標準 IEEE 754
	// 收到: data[0] data[1] data[2] data[3] (對應 Word1_High Word1_Low Word2_High Word2_Low)
	// 需要重排為: data[2] data[3] data[0] data[1] (標準 IEEE 754)

	ieeeBytes := make([]byte, 4)
	ieeeBytes[0] = data[2] // Byte 2
	ieeeBytes[1] = data[3] // Byte 3
	ieeeBytes[2] = data[0] // Byte 0
	ieeeBytes[3] = data[1] // Byte 1

	// 轉換為 float32
	bits := binary.BigEndian.Uint32(ieeeBytes)
	pressure := math.Float32frombits(bits)

	pm.logger.Printf("浮點數解析: 原始=%02X%02X%02X%02X, 重排=%02X%02X%02X%02X, 值=%.2f",
		data[0], data[1], data[2], data[3],
		ieeeBytes[0], ieeeBytes[1], ieeeBytes[2], ieeeBytes[3],
		pressure)

	return float64(pressure)
}

// GetReadings 獲取讀數通道
func (pm *PressureMeter) GetReadings() <-chan PressureReading {
	return pm.readings
}

// Close 關閉連接
func (pm *PressureMeter) Close() error {
	pm.Stop()

	// 關閉 Modbus 連接
	if pm.handler != nil {
		return pm.handler.Close()
	}

	return nil
}

// SetDataFormat 設置數據格式
func (pm *PressureMeter) SetDataFormat(format DataFormatType) {
	pm.dataFormat = format
	pm.logger.Printf("數據格式已設置為: %d", format)
}

// GetStatus 獲取設備狀態
func (pm *PressureMeter) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":        pm.running,
		"slave_id":       pm.slaveID,
		"data_format":    pm.dataFormat,
		"queue_size":     len(pm.readings),
		"queue_capacity": cap(pm.readings),
	}
}

// IsRunning 檢查設備是否正在運行
func (pm *PressureMeter) IsRunning() bool {
	return pm.running
}

// GetSlaveID 獲取從站ID
func (pm *PressureMeter) GetSlaveID() byte {
	return pm.slaveID
}

// GetDataFormat 獲取數據格式
func (pm *PressureMeter) GetDataFormat() DataFormatType {
	return pm.dataFormat
}

// TestConnection 測試連接是否正常
func (pm *PressureMeter) TestConnection() error {
	reading := pm.ReadPressure()
	if !reading.Valid {
		return fmt.Errorf("連接測試失敗: %s", reading.Error)
	}
	pm.logger.Printf("連接測試成功，當前壓力: %.2f Pa", reading.Pressure)
	return nil
}

// GetLastReading 獲取最後一次讀數（非阻塞）
func (pm *PressureMeter) GetLastReading() *PressureReading {
	select {
	case reading := <-pm.readings:
		return &reading
	default:
		return nil
	}
}

// FlushReadings 清空讀數緩衝區
func (pm *PressureMeter) FlushReadings() int {
	count := 0
	for {
		select {
		case <-pm.readings:
			count++
		default:
			pm.logger.Printf("已清空 %d 個緩衝讀數", count)
			return count
		}
	}
}

// String 實現 Stringer 接口，方便打印設備信息
func (pm *PressureMeter) String() string {
	status := "停止"
	if pm.running {
		status = "運行中"
	}

	formatStr := "十進制"
	if pm.dataFormat == FloatFormat {
		formatStr = "浮點數"
	}

	return fmt.Sprintf("壓差儀[站點:%d, 格式:%s, 狀態:%s]",
		pm.slaveID, formatStr, status)
}
