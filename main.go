// main.go - 壓差儀監測程式主入口
package main

import (
	"Pushi_Pressure_Meter/pressure"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// AppInfo 應用程式信息
type AppInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	BuildTime   string `json:"build_time"`
}

// 應用程式信息
var appInfo = AppInfo{
	Name:        "壓差儀監測工具",
	Version:     "1.0.0",
	Description: "普時達壓差儀 RS485 監測和數據採集工具",
	Author:      "Foyliu <s225002731@gmail.com>",
	BuildTime:   "2025-06-23", // 編譯時會替換
}

// 命令列參數
var (
	showVersion    = flag.Bool("version", false, "顯示版本信息")
	showHelp       = flag.Bool("help", false, "顯示幫助信息")
	autoScan       = flag.Bool("auto-scan", false, "自動掃描並配置第一個找到的設備")
	quickScan      = flag.Bool("quick-scan", false, "快速掃描設備")
	fullScan       = flag.Bool("full-scan", false, "完整掃描設備")
	testConfig     = flag.Bool("test-config", false, "測試配置並退出")
	generateConfig = flag.Bool("generate-config", false, "生成配置檔案示例")
	daemon         = flag.Bool("daemon", false, "以守護程序模式運行")
	logFile        = flag.String("log", "", "日誌檔案路徑")
	configFile     = flag.String("config", "", "指定配置檔案路徑")
	outputFormat   = flag.String("output", "text", "輸出格式 (text/json/csv)")
	maxReadings    = flag.Int("max-readings", 0, "最大讀數數量，0為無限制")
	duration       = flag.Duration("duration", 0, "運行時間，0為無限制")
	verbose        = flag.Bool("verbose", false, "詳細輸出")
	quiet          = flag.Bool("quiet", false, "靜默模式")
)

func main() {
	// 解析命令列參數
	flag.Parse()

	// 設置日誌
	logger := setupLogger()

	// 處理特殊命令
	if *showVersion {
		printVersion()
		return
	}

	if *showHelp {
		printHelp()
		return
	}

	if *generateConfig {
		generateConfigFiles()
		return
	}

	// 打印啟動信息
	if !*quiet {
		printStartupBanner(logger)
	}

	// 根據不同的模式運行
	switch {
	case *autoScan:
		runAutoScanMode(logger)
	case *quickScan:
		runQuickScanMode(logger)
	case *fullScan:
		runFullScanMode(logger)
	case *testConfig:
		runTestConfigMode(logger)
	default:
		runNormalMode(logger)
	}
}

// setupLogger 設置日誌記錄器
func setupLogger() *log.Logger {
	var logger *log.Logger

	if *logFile != "" {
		// 創建日誌目錄
		dir := filepath.Dir(*logFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("❌ 創建日誌目錄失敗: %v", err)
		}

		// 打開日誌檔案
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("❌ 打開日誌檔案失敗: %v", err)
		}

		logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
		fmt.Printf("📝 日誌將寫入: %s\n", *logFile)
	} else {
		logger = log.Default()
	}

	// 設置日誌級別
	if *quiet {
		logger.SetOutput(os.Stderr) // 靜默模式下只輸出錯誤
	} else if *verbose {
		logger.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	}

	return logger
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("%s v%s\n", appInfo.Name, appInfo.Version)
	fmt.Printf("構建時間: %s\n", appInfo.BuildTime)
	fmt.Printf("作者: %s\n", appInfo.Author)
}

// printStartupBanner 打印啟動橫幅
func printStartupBanner(logger *log.Logger) {
	// 計算內容長度以確保對齊
	titleLine := fmt.Sprintf("🌡️  %s v%s", appInfo.Name, appInfo.Version)
	buildLine := fmt.Sprintf("📅 構建時間: %s", appInfo.BuildTime)
	authorLine := fmt.Sprintf("👤 作者: %s", appInfo.Author)

	// 找出最長的行來確定邊框寬度
	maxWidth := 0
	lines := []string{
		titleLine,
		"📡 普時達壓差儀 RS485 監測工具",
		"🔧 支援自動掃描和多種數據格式",
		buildLine,
		authorLine,
	}

	for _, line := range lines {
		// 計算實際顯示寬度（考慮 emoji 和中文字符）
		width := calculateDisplayWidth(line)
		if width > maxWidth {
			maxWidth = width
		}
	}

	// 確保最小寬度
	if maxWidth < 50 {
		maxWidth = 50
	}

	// 構建橫幅
	border := "═"
	padding := 2
	totalWidth := maxWidth + padding*2

	banner := fmt.Sprintf(`
╔%s╗
║ %-*s ║
║ %-*s ║
║ %-*s ║
║%s║
║ %-*s ║
║ %-*s ║
╚%s╝
`,
		strings.Repeat(border, totalWidth),
		maxWidth, titleLine,
		maxWidth, "📡 普時達壓差儀 RS485 監測工具",
		maxWidth, "🔧 支援自動掃描和多種數據格式",
		strings.Repeat("─", totalWidth),
		maxWidth, buildLine,
		maxWidth, authorLine,
		strings.Repeat(border, totalWidth),
	)

	fmt.Print(banner)
	logger.Printf("程式啟動: %s v%s", appInfo.Name, appInfo.Version)
}

// calculateDisplayWidth 計算字符串的實際顯示寬度
func calculateDisplayWidth(s string) int {
	width := 0
	runes := []rune(s)

	for _, r := range runes {
		if r < 128 {
			// ASCII 字符寬度為 1
			width++
		} else {
			// 中文字符和 emoji 寬度為 2
			width += 2
		}
	}
	return width
}

// printHelp 打印幫助信息
func printHelp() {
	fmt.Printf("%s v%s\n\n", appInfo.Name, appInfo.Version)
	fmt.Println("🔧 壓差儀監測工具 - 支援普時達壓差儀 RS485 通信")
	fmt.Println()
	fmt.Println("使用方法:")
	fmt.Printf("  %s [選項]\n\n", os.Args[0])

	fmt.Println("📊 掃描模式:")
	fmt.Println("  --auto-scan      自動掃描並配置第一個找到的設備")
	fmt.Println("  --quick-scan     快速掃描常用設備配置")
	fmt.Println("  --full-scan      完整掃描所有可能的設備")
	fmt.Println()

	fmt.Println("⚙️  配置選項:")
	fmt.Println("  --config FILE    指定配置檔案路徑")
	fmt.Println("  --generate-config 生成配置檔案示例")
	fmt.Println("  --test-config    測試配置並退出")
	fmt.Println()

	fmt.Println("📝 輸出選項:")
	fmt.Println("  --output FORMAT  輸出格式 (text/json/csv)")
	fmt.Println("  --log FILE       指定日誌檔案路徑")
	fmt.Println("  --verbose        詳細輸出")
	fmt.Println("  --quiet          靜默模式")
	fmt.Println()

	fmt.Println("🎮 控制選項:")
	fmt.Println("  --max-readings N 最大讀數數量")
	fmt.Println("  --duration TIME  運行時間 (如: 30s, 5m, 1h)")
	fmt.Println("  --daemon         守護程序模式")
	fmt.Println()

	fmt.Println("ℹ️  信息選項:")
	fmt.Println("  --version        顯示版本信息")
	fmt.Println("  --help           顯示此幫助信息")
	fmt.Println()

	fmt.Println("📖 配置方式:")
	fmt.Println("  1. 環境變數:")
	fmt.Println("     export PRESSURE_DEVICE=/dev/ttyUSB0")
	fmt.Println("     export PRESSURE_SLAVE_ID=22")
	fmt.Println("     export PRESSURE_READ_INTERVAL=1s")
	fmt.Println("     export PRESSURE_DATA_FORMAT=decimal")
	fmt.Println()

	fmt.Println("  2. 配置檔案 (pressure_config.yaml):")
	fmt.Println("     device: /dev/ttyUSB0")
	fmt.Println("     slaveid: 22")
	fmt.Println("     readinterval: 1s")
	fmt.Println("     dataformat: 0")
	fmt.Println()

	fmt.Println("  3. 命令列參數:")
	fmt.Println("     --device=/dev/ttyUSB0 --slave-id=22 --interval=1s")
	fmt.Println()

	fmt.Println("💡 使用示例:")
	fmt.Println("  # 自動掃描並開始監測")
	fmt.Printf("  %s --auto-scan\n", os.Args[0])
	fmt.Println()
	fmt.Println("  # 快速掃描設備")
	fmt.Printf("  %s --quick-scan\n", os.Args[0])
	fmt.Println()
	fmt.Println("  # 使用指定配置監測 5 分鐘")
	fmt.Printf("  %s --config=my_config.yaml --duration=5m\n", os.Args[0])
	fmt.Println()
	fmt.Println("  # JSON 格式輸出到檔案")
	fmt.Printf("  %s --output=json --log=pressure.log\n", os.Args[0])
	fmt.Println()
	fmt.Println("  # 守護程序模式")
	fmt.Printf("  %s --daemon --log=/var/log/pressure.log\n", os.Args[0])
}

// runAutoScanMode 自動掃描模式
func runAutoScanMode(logger *log.Logger) {
	fmt.Println("🔍 開始自動掃描壓差儀設備...")

	scanner := pressure.NewScanner(logger).SetVerbose(!*quiet)
	config, err := scanner.AutoConfigure()
	if err != nil {
		logger.Fatalf("❌ 自動配置失敗: %v", err)
	}

	fmt.Printf("✅ 自動配置成功！\n")
	fmt.Printf("   📍 設備: %s\n", config.Device)
	fmt.Printf("   🎯 站點號: %d (0x%02X)\n", config.SlaveID, config.SlaveID)
	fmt.Printf("   📊 數據格式: %s\n", config.DataFormat)
	fmt.Printf("   ⏱️  讀取間隔: %v\n", config.ReadInterval)

	// 開始監測
	startMonitoring(config, logger)
}

// runQuickScanMode 快速掃描模式
func runQuickScanMode(logger *log.Logger) {
	fmt.Println("⚡ 開始快速掃描...")

	scanner := pressure.NewScanner(logger).SetVerbose(!*quiet)
	result, err := scanner.QuickScan()
	if err != nil {
		logger.Fatalf("❌ 掃描失敗: %v", err)
	}

	scanner.PrintScanResults(result)

	// 如果找到設備，讓用戶選擇
	responsiveDevices := getResponsiveDevices(result.Devices)
	if len(responsiveDevices) == 0 {
		fmt.Println("❌ 未找到任何響應設備")
		return
	}

	// 使用第一個設備開始監測
	device := responsiveDevices[0]
	config := createConfigFromDevice(device, logger)

	fmt.Printf("\n🚀 使用設備: %s (站點 %d) 開始監測\n",
		device.Device, device.SlaveID)
	startMonitoring(config, logger)
}

// runFullScanMode 完整掃描模式
func runFullScanMode(logger *log.Logger) {
	fmt.Println("🔍 開始完整掃描...")

	scanner := pressure.NewScanner(logger).SetVerbose(!*quiet)
	result, err := scanner.FullScan()
	if err != nil {
		logger.Fatalf("❌ 掃描失敗: %v", err)
	}

	scanner.PrintScanResults(result)

	// 保存掃描結果
	if err := saveScanResults(result); err != nil {
		logger.Printf("⚠️  保存掃描結果失敗: %v", err)
	}
}

// runTestConfigMode 測試配置模式
func runTestConfigMode(logger *log.Logger) {
	fmt.Println("🧪 測試配置...")

	loader := pressure.NewConfigLoader()
	if *configFile != "" {
		loader.SetConfigFile(*configFile)
	}

	info, err := loader.LoadConfigWithSource()
	if err != nil {
		logger.Fatalf("❌ 載入配置失敗: %v", err)
	}

	fmt.Println("✅ 配置載入成功!")
	loader.PrintConfigWithSource(info)

	// 測試設備連接
	fmt.Println("\n🔌 測試設備連接...")
	pm, err := pressure.NewPressureMeter(*info.Config)
	if err != nil {
		logger.Fatalf("❌ 創建設備失敗: %v", err)
	}
	defer pm.Close()

	if err := pm.TestConnection(); err != nil {
		logger.Fatalf("❌ 設備連接測試失敗: %v", err)
	}

	fmt.Println("✅ 設備連接測試成功!")

	// 讀取一次數據
	reading := pm.ReadPressure()
	if reading.Valid {
		fmt.Printf("📊 當前壓力: %.2f Pa\n", reading.Pressure)
	} else {
		fmt.Printf("❌ 讀取壓力失敗: %s\n", reading.Error)
	}
}

// runNormalMode 正常模式
func runNormalMode(logger *log.Logger) {
	fmt.Println("📋 載入配置...")

	loader := pressure.NewConfigLoader()
	if *configFile != "" {
		loader.SetConfigFile(*configFile)
	}

	config, err := loader.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 載入配置失敗: %v\n", err)
		fmt.Println("\n💡 建議:")
		fmt.Println("   - 使用 --auto-scan 自動掃描設備")
		fmt.Println("   - 使用 --quick-scan 快速掃描")
		fmt.Println("   - 使用 --generate-config 生成配置檔案")
		fmt.Println("   - 設置環境變數或創建配置文件")
		fmt.Println("   - 使用 --help 查看詳細幫助")
		return
	}

	if !*quiet {
		loader.PrintConfig(config)
	}

	startMonitoring(config, logger)
}

// startMonitoring 開始監測壓力
func startMonitoring(config *pressure.Config, logger *log.Logger) {
	fmt.Println("🚀 啟動壓差儀監測...")

	// 創建壓差儀實例
	pm, err := pressure.NewPressureMeter(*config)
	if err != nil {
		logger.Fatalf("❌ 創建壓差儀失敗: %v", err)
	}
	defer pm.Close()

	// 測試連接
	if err := pm.TestConnection(); err != nil {
		logger.Fatalf("❌ 設備連接失敗: %v", err)
	}

	// 創建上下文和取消函數
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 如果設置了運行時間限制
	if *duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

	// 開始讀取
	pm.Start(config.ReadInterval)

	// 創建信號通道，用於優雅關閉
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if !*quiet {
		fmt.Println("📊 開始實時監測壓力數據...")
		if *duration > 0 {
			fmt.Printf("⏰ 運行時間: %v\n", *duration)
		}
		if *maxReadings > 0 {
			fmt.Printf("📈 最大讀數: %d\n", *maxReadings)
		}
		fmt.Println("   按 Ctrl+C 停止監測")
		fmt.Println()
	}

	// 統計信息
	stats := &pressure.Statistics{}
	readingCount := 0

	// 處理讀數
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case reading := <-pm.GetReadings():
				readingCount++

				if reading.Valid {
					stats.Update(reading.Pressure)
					outputReading(reading, readingCount, stats)
				} else {
					outputError(reading, readingCount)
				}

				// 檢查是否達到最大讀數
				if *maxReadings > 0 && readingCount >= *maxReadings {
					logger.Printf("已達到最大讀數限制: %d", *maxReadings)
					cancel()
					return
				}
			}
		}
	}()

	// 等待退出信號或超時
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("\n⏰ 已達到運行時間限制: %v\n", *duration)
		}
	case sig := <-sigChan:
		fmt.Printf("\n🛑 接收到信號: %v\n", sig)
	}

	fmt.Println("🛑 正在停止監測...")
	pm.Stop()

	// 打印統計信息
	if !*quiet && readingCount > 0 {
		fmt.Println("\n📊 監測統計:")
		fmt.Printf("   📈 總讀數: %d\n", readingCount)
		fmt.Printf("   ⏱️  運行時間: %v\n", time.Since(stats.LastTime.Add(-time.Duration(readingCount)*config.ReadInterval)))
		fmt.Printf("   📊 %s\n", stats)
	}

	fmt.Println("✅ 監測已停止")
}

// outputReading 輸出壓力讀數
func outputReading(reading pressure.PressureReading, count int, stats *pressure.Statistics) {
	timestamp := reading.Timestamp.Format("15:04:05")

	switch *outputFormat {
	case "json":
		data := map[string]interface{}{
			"timestamp": reading.Timestamp,
			"count":     count,
			"slave_id":  reading.SlaveID,
			"pressure":  reading.Pressure,
			"unit":      "Pa",
			"valid":     reading.Valid,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Println(string(jsonData))

	case "csv":
		if count == 1 {
			fmt.Println("timestamp,count,slave_id,pressure,unit,valid")
		}
		fmt.Printf("%s,%d,%d,%.3f,Pa,%t\n",
			reading.Timestamp.Format("2006-01-02 15:04:05"),
			count, reading.SlaveID, reading.Pressure, reading.Valid)

	default: // text
		if !*quiet {
			fmt.Printf("[%s] #%d 站點%d: %.2f Pa (平均: %.2f Pa)\n",
				timestamp, count, reading.SlaveID, reading.Pressure, stats.Mean)
		}
	}
}

// outputError 輸出錯誤信息
func outputError(reading pressure.PressureReading, count int) {
	timestamp := reading.Timestamp.Format("15:04:05")

	switch *outputFormat {
	case "json":
		data := map[string]interface{}{
			"timestamp": reading.Timestamp,
			"count":     count,
			"slave_id":  reading.SlaveID,
			"error":     reading.Error,
			"valid":     false,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Println(string(jsonData))

	case "csv":
		fmt.Printf("%s,%d,%d,NaN,Pa,false\n",
			reading.Timestamp.Format("2006-01-02 15:04:05"),
			count, reading.SlaveID)

	default: // text
		fmt.Printf("[%s] #%d ❌ 讀取失敗: %s\n",
			timestamp, count, reading.Error)
	}
}

// generateConfigFiles 生成配置檔案示例
func generateConfigFiles() {
	fmt.Println("📝 生成配置檔案示例...")

	// 生成 YAML 配置
	yamlConfig := `# 壓差儀配置檔案 (YAML 格式)
device: /dev/ttyUSB0          # RS485 設備路徑
slaveid: 22                   # 從站ID (1-247)
readinterval: 1s              # 讀取間隔
dataformat: 0                 # 數據格式: 0=十進制, 1=浮點數
`

	// 生成 JSON 配置
	jsonConfig := `{
  "device": "/dev/ttyUSB0",
  "slaveid": 22,
  "readinterval": "1s",
  "dataformat": 0
}`

	// 保存檔案
	files := map[string]string{
		"pressure_config.yaml": yamlConfig,
		"pressure_config.json": jsonConfig,
	}

	for filename, content := range files {
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			fmt.Printf("❌ 創建 %s 失敗: %v\n", filename, err)
		} else {
			fmt.Printf("✅ 已創建: %s\n", filename)
		}
	}

	fmt.Println("\n📖 配置說明:")
	fmt.Println("  device: RS485 設備路徑")
	fmt.Println("    Linux: /dev/ttyUSB0, /dev/ttyACM0")
	fmt.Println("    Windows: COM1, COM2")
	fmt.Println("  slaveid: Modbus 從站ID (1-247)")
	fmt.Println("  readinterval: 讀取間隔 (如: 1s, 500ms, 2m)")
	fmt.Println("  dataformat: 0=十進制(預設), 1=浮點數")
}

// 輔助函數

// getResponsiveDevices 獲取響應的設備
func getResponsiveDevices(devices []pressure.DeviceInfo) []pressure.DeviceInfo {
	var responsive []pressure.DeviceInfo
	for _, device := range devices {
		if device.Responsive {
			responsive = append(responsive, device)
		}
	}
	return responsive
}

// createConfigFromDevice 從設備信息創建配置
func createConfigFromDevice(device pressure.DeviceInfo, logger *log.Logger) *pressure.Config {
	return &pressure.Config{
		Device:       device.Device,
		SlaveID:      device.SlaveID,
		ReadInterval: time.Second,
		DataFormat:   device.DataFormat,
		Logger:       logger,
	}
}

// saveScanResults 保存掃描結果
func saveScanResults(result *pressure.ScanResult) error {
	filename := fmt.Sprintf("scan_results_%s.json",
		time.Now().Format("20060102_150405"))

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	fmt.Printf("💾 掃描結果已保存到: %s\n", filename)
	return nil
}
