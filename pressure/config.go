// pressure/config.go - 壓差儀配置管理
package pressure

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigLoader 配置加載器
type ConfigLoader struct {
	configFile string
	useEnv     bool
	useFlags   bool
}

// ConfigSource 配置來源類型
type ConfigSource int

const (
	SourceDefault ConfigSource = iota // 默認值
	SourceFile                        // 配置文件
	SourceEnv                         // 環境變數
	SourceFlags                       // 命令列參數
)

// ConfigInfo 配置信息，包含來源追蹤
type ConfigInfo struct {
	Config *Config                 `json:"config"`
	Source map[string]ConfigSource `json:"source"` // 每個字段的來源
}

// NewConfigLoader 創建配置加載器
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		useEnv:   true,
		useFlags: true,
	}
}

// SetConfigFile 設置配置文件路徑
func (cl *ConfigLoader) SetConfigFile(path string) *ConfigLoader {
	cl.configFile = path
	return cl
}

// SetUseEnv 設置是否使用環境變數
func (cl *ConfigLoader) SetUseEnv(use bool) *ConfigLoader {
	cl.useEnv = use
	return cl
}

// SetUseFlags 設置是否使用命令列參數
func (cl *ConfigLoader) SetUseFlags(use bool) *ConfigLoader {
	cl.useFlags = use
	return cl
}

// LoadConfig 加載配置，優先級：命令列 > 環境變數 > 配置檔案 > 默認值
func (cl *ConfigLoader) LoadConfig() (*Config, error) {
	info, err := cl.LoadConfigWithSource()
	if err != nil {
		return nil, err
	}
	return info.Config, nil
}

// LoadConfigWithSource 加載配置並追蹤來源
func (cl *ConfigLoader) LoadConfigWithSource() (*ConfigInfo, error) {
	info := &ConfigInfo{
		Config: &Config{},
		Source: make(map[string]ConfigSource),
	}

	// 1. 設置默認值
	cl.setDefaults(info)

	// 2. 從配置檔案讀取（如果存在）
	if err := cl.loadFromFile(info); err != nil {
		log.Printf("警告：讀取配置檔案失敗: %v", err)
	}

	// 3. 從環境變數讀取
	if cl.useEnv {
		cl.loadFromEnv(info)
	}

	// 4. 從命令列參數讀取（最高優先級）
	if cl.useFlags {
		cl.loadFromFlags(info)
	}

	// 5. 驗證配置
	if err := cl.validateConfig(info.Config); err != nil {
		return nil, fmt.Errorf("配置驗證失敗: %v", err)
	}

	return info, nil
}

// setDefaults 設置默認值
func (cl *ConfigLoader) setDefaults(info *ConfigInfo) {
	// 根據操作系統設置默認設備路徑
	defaultDevice := "/dev/ttyUSB0" // Linux 默認
	if isWindows() {
		defaultDevice = "COM1" // Windows 默認
	}

	info.Config.Device = defaultDevice
	info.Config.SlaveID = 0x16                 // 默認站點號 22
	info.Config.ReadInterval = 1 * time.Second // 默認讀取間隔
	info.Config.DataFormat = DecimalFormat     // 默認十進制格式
	info.Config.Logger = log.Default()

	// 記錄來源
	info.Source["device"] = SourceDefault
	info.Source["slaveid"] = SourceDefault
	info.Source["readinterval"] = SourceDefault
	info.Source["dataformat"] = SourceDefault
}

// loadFromFile 從配置檔案讀取
func (cl *ConfigLoader) loadFromFile(info *ConfigInfo) error {
	// 按優先級檢查配置檔案
	configFiles := []string{
		"pressure_config.yaml",
		"pressure_config.yml",
		"pressure_config.json",
		"config.yaml",
		"config.yml",
		"config.json",
	}

	// 檢查常見的配置目錄
	configDirs := []string{
		"./",
		"./config/",
		"/etc/pressure/",
		"/usr/local/etc/pressure/",
	}

	// 如果指定了配置檔案，優先使用
	if cl.configFile != "" {
		configFiles = []string{cl.configFile}
		configDirs = []string{"./"}
	}

	var lastErr error
	for _, dir := range configDirs {
		for _, filename := range configFiles {
			fullPath := dir + filename
			if err := cl.loadConfigFile(fullPath, info); err == nil {
				log.Printf("已載入配置檔案: %s", fullPath)
				return nil
			} else {
				lastErr = err
			}
		}
	}

	return fmt.Errorf("未找到有效的配置檔案，最後錯誤: %v", lastErr)
}

// loadConfigFile 載入指定的配置檔案
func (cl *ConfigLoader) loadConfigFile(filename string, info *ConfigInfo) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("檔案不存在: %s", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("讀取檔案失敗: %v", err)
	}

	// 創建臨時配置來解析檔案
	tempConfig := &Config{}

	// 根據副檔名選擇解析方式
	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".yaml") ||
		strings.HasSuffix(strings.ToLower(filename), ".yml"):
		err = yaml.Unmarshal(data, tempConfig)
	case strings.HasSuffix(strings.ToLower(filename), ".json"):
		err = json.Unmarshal(data, tempConfig)
	default:
		return fmt.Errorf("不支援的檔案格式: %s", filename)
	}

	if err != nil {
		return fmt.Errorf("解析配置檔案失敗: %v", err)
	}

	// 將檔案中的配置合併到主配置中
	cl.mergeConfig(info, tempConfig, SourceFile)
	return nil
}

// mergeConfig 合併配置並記錄來源
func (cl *ConfigLoader) mergeConfig(info *ConfigInfo, source *Config, sourceType ConfigSource) {
	if source.Device != "" {
		info.Config.Device = source.Device
		info.Source["device"] = sourceType
	}
	if source.SlaveID != 0 {
		info.Config.SlaveID = source.SlaveID
		info.Source["slaveid"] = sourceType
	}
	if source.ReadInterval != 0 {
		info.Config.ReadInterval = source.ReadInterval
		info.Source["readinterval"] = sourceType
	}
	// DataFormat 可以是 0，所以需要特殊處理
	info.Config.DataFormat = source.DataFormat
	info.Source["dataformat"] = sourceType
}

// loadFromEnv 從環境變數讀取
func (cl *ConfigLoader) loadFromEnv(info *ConfigInfo) {
	// 設備路徑
	if device := os.Getenv("PRESSURE_DEVICE"); device != "" {
		info.Config.Device = device
		info.Source["device"] = SourceEnv
	}

	// 站點號
	if slaveIDStr := os.Getenv("PRESSURE_SLAVE_ID"); slaveIDStr != "" {
		if slaveID, err := parseSlaveID(slaveIDStr); err == nil {
			info.Config.SlaveID = slaveID
			info.Source["slaveid"] = SourceEnv
		} else {
			log.Printf("警告：環境變數 PRESSURE_SLAVE_ID 格式錯誤: %v", err)
		}
	}

	// 讀取間隔
	if intervalStr := os.Getenv("PRESSURE_READ_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			info.Config.ReadInterval = interval
			info.Source["readinterval"] = SourceEnv
		} else {
			log.Printf("警告：環境變數 PRESSURE_READ_INTERVAL 格式錯誤: %v", err)
		}
	}

	// 數據格式
	if formatStr := os.Getenv("PRESSURE_DATA_FORMAT"); formatStr != "" {
		if format, err := parseDataFormat(formatStr); err == nil {
			info.Config.DataFormat = format
			info.Source["dataformat"] = SourceEnv
		} else {
			log.Printf("警告：環境變數 PRESSURE_DATA_FORMAT 格式錯誤: %v", err)
		}
	}

	log.Println("已載入環境變數配置")
}

// loadFromFlags 從命令列參數讀取
func (cl *ConfigLoader) loadFromFlags(info *ConfigInfo) {
	// 只有在 flag 還沒有被解析時才定義參數
	if !flag.Parsed() {
		device := flag.String("device", info.Config.Device, "RS485 設備路徑")
		slaveID := flag.Uint("slave-id", uint(info.Config.SlaveID), "Modbus 站點號 (1-247)")
		interval := flag.Duration("interval", info.Config.ReadInterval, "讀取間隔時間")
		format := flag.String("format", "decimal", "數據格式 (decimal/float)")
		configFile := flag.String("config", "", "配置檔案路徑")

		flag.Parse()

		// 更新配置
		if *device != info.Config.Device {
			info.Config.Device = *device
			info.Source["device"] = SourceFlags
		}
		if byte(*slaveID) != info.Config.SlaveID {
			info.Config.SlaveID = byte(*slaveID)
			info.Source["slaveid"] = SourceFlags
		}
		if *interval != info.Config.ReadInterval {
			info.Config.ReadInterval = *interval
			info.Source["readinterval"] = SourceFlags
		}

		// 處理數據格式
		if parsedFormat, err := parseDataFormat(*format); err == nil {
			if parsedFormat != info.Config.DataFormat {
				info.Config.DataFormat = parsedFormat
				info.Source["dataformat"] = SourceFlags
			}
		}

		// 設置配置檔案路徑
		if *configFile != "" {
			cl.configFile = *configFile
		}
	}

	log.Println("已載入命令列參數配置")
}

// validateConfig 驗證配置
func (cl *ConfigLoader) validateConfig(config *Config) error {
	if config.Device == "" {
		return fmt.Errorf("設備路徑不能為空")
	}

	if config.SlaveID < 1 || config.SlaveID > 247 {
		return fmt.Errorf("站點號必須在 1-247 之間，當前: %d", config.SlaveID)
	}

	if config.ReadInterval < 100*time.Millisecond {
		return fmt.Errorf("讀取間隔不能小於 100ms，當前: %v", config.ReadInterval)
	}

	// 檢查設備路徑是否存在（僅在類 Unix 系統上）
	if !isWindows() {
		if _, err := os.Stat(config.Device); os.IsNotExist(err) {
			log.Printf("警告：設備路徑可能不存在: %s", config.Device)
		}
	}

	return nil
}

// SaveConfig 保存配置到檔案
func (cl *ConfigLoader) SaveConfig(config *Config, filename string) error {
	var data []byte
	var err error

	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".yaml") ||
		strings.HasSuffix(strings.ToLower(filename), ".yml"):
		data, err = yaml.Marshal(config)
	case strings.HasSuffix(strings.ToLower(filename), ".json"):
		data, err = json.MarshalIndent(config, "", "  ")
	default:
		return fmt.Errorf("不支援的檔案格式，請使用 .yaml 或 .json")
	}

	if err != nil {
		return fmt.Errorf("序列化配置失敗: %v", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// PrintConfig 打印當前配置
func (cl *ConfigLoader) PrintConfig(config *Config) {
	fmt.Println("=== 壓差儀配置 ===")
	fmt.Printf("設備路徑: %s\n", config.Device)
	fmt.Printf("站點號: %d (0x%02X)\n", config.SlaveID, config.SlaveID)
	fmt.Printf("讀取間隔: %v\n", config.ReadInterval)
	fmt.Printf("數據格式: %s\n", formatToString(config.DataFormat))
	fmt.Println("==================")
}

// PrintConfigWithSource 打印配置及其來源
func (cl *ConfigLoader) PrintConfigWithSource(info *ConfigInfo) {
	fmt.Println("=== 壓差儀配置（含來源）===")
	fmt.Printf("設備路徑: %s [%s]\n", info.Config.Device, sourceToString(info.Source["device"]))
	fmt.Printf("站點號: %d (0x%02X) [%s]\n", info.Config.SlaveID, info.Config.SlaveID, sourceToString(info.Source["slaveid"]))
	fmt.Printf("讀取間隔: %v [%s]\n", info.Config.ReadInterval, sourceToString(info.Source["readinterval"]))
	fmt.Printf("數據格式: %s [%s]\n", formatToString(info.Config.DataFormat), sourceToString(info.Source["dataformat"]))
	fmt.Println("========================")
}

// GenerateConfigExample 生成配置檔案示例
func GenerateConfigExample() {
	config := &Config{
		Device:       "/dev/ttyUSB0",
		SlaveID:      22,
		ReadInterval: 1 * time.Second,
		DataFormat:   DecimalFormat,
	}

	fmt.Println("=== YAML 配置檔案示例 (pressure_config.yaml) ===")
	yamlData, _ := yaml.Marshal(config)
	fmt.Println(string(yamlData))

	fmt.Println("=== JSON 配置檔案示例 (pressure_config.json) ===")
	jsonData, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println(string(jsonData))
}

// PrintEnvExample 打印環境變數示例
func PrintEnvExample() {
	fmt.Println("=== 環境變數設置示例 ===")
	fmt.Println("export PRESSURE_DEVICE=/dev/ttyUSB0")
	fmt.Println("export PRESSURE_SLAVE_ID=22")
	fmt.Println("export PRESSURE_READ_INTERVAL=1s")
	fmt.Println("export PRESSURE_DATA_FORMAT=decimal")
	fmt.Println("========================")
}

// PrintDockerExample 打印 Docker 環境變數示例
func PrintDockerExample() {
	fmt.Println("=== Docker 環境變數示例 ===")
	fmt.Println("docker run -d \\")
	fmt.Println("  --device=/dev/ttyUSB0 \\")
	fmt.Println("  -e PRESSURE_DEVICE=/dev/ttyUSB0 \\")
	fmt.Println("  -e PRESSURE_SLAVE_ID=22 \\")
	fmt.Println("  -e PRESSURE_READ_INTERVAL=2s \\")
	fmt.Println("  -e PRESSURE_DATA_FORMAT=decimal \\")
	fmt.Println("  pressure-meter-macArm64:latest")
	fmt.Println("==========================")
}

// 輔助函數

// parseSlaveID 解析站點號，支援十進制和十六進制
func parseSlaveID(s string) (byte, error) {
	s = strings.TrimSpace(s)

	// 支援十六進制格式 (0x16, 0X16)
	if strings.HasPrefix(strings.ToLower(s), "0x") {
		if val, err := strconv.ParseUint(s, 0, 8); err == nil {
			return byte(val), nil
		}
	}

	// 十進制格式
	if val, err := strconv.ParseUint(s, 10, 8); err == nil {
		return byte(val), nil
	}

	return 0, fmt.Errorf("無效的站點號格式: %s", s)
}

// parseDataFormat 解析數據格式
func parseDataFormat(s string) (DataFormatType, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "decimal", "dec", "0":
		return DecimalFormat, nil
	case "float", "floating", "1":
		return FloatFormat, nil
	default:
		return DecimalFormat, fmt.Errorf("無效的數據格式: %s", s)
	}
}

// formatToString 將數據格式轉為字符串
func formatToString(format DataFormatType) string {
	switch format {
	case DecimalFormat:
		return "十進制"
	case FloatFormat:
		return "浮點數"
	default:
		return "未知"
	}
}

// sourceToString 將配置來源轉為字符串
func sourceToString(source ConfigSource) string {
	switch source {
	case SourceDefault:
		return "默認"
	case SourceFile:
		return "檔案"
	case SourceEnv:
		return "環境變數"
	case SourceFlags:
		return "命令列"
	default:
		return "未知"
	}
}

// isWindows 檢查是否為 Windows 系統
func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
}
