# 🌡️ 壓差儀監測工具


普時達壓差儀 RS485 監測和數據採集工具，支援自動設備掃描、多種數據格式和靈活的配置方式。

## ✨ 特色功能

- 🔍 **自動設備掃描** - 智能發現和配置壓差儀設備
- 📊 **多種數據格式** - 支援十進制和 IEEE 754 浮點數格式
- 🌐 **跨平台支援** - Linux、Windows、macOS 全平台支援
- 📈 **實時監測** - 連續監測壓力數據並提供統計分析
- 🔧 **靈活配置** - 支援環境變數、配置檔案、命令列參數
- 📝 **多種輸出** - 文本、JSON、CSV 格式輸出

## 📋 目錄

- [快速開始](#-快速開始)
- [安裝方式](#-安裝方式)
- [使用方法](#-使用方法)
- [配置說明](#-配置說明)
- [API 文檔](#-api-文檔)
- [故障排除](#-故障排除)
- [開發指南](#-開發指南)
- [貢獻指南](#-貢獻指南)

## 🚀 快速開始

### 方式一：自動掃描（推薦）

```bash
# 下載並運行
./pressure-meter --auto-scan
```

### 方式二：手動配置

```bash
# 設置環境變數
export PRESSURE_DEVICE=/dev/ttyUSB0
export PRESSURE_SLAVE_ID=22

# 運行程式
./pressure-meter
```

### 方式三：使用配置檔案

```bash
# 生成配置檔案
./pressure-meter --generate-config

# 編輯配置
nano pressure_config.yaml

# 運行程式
./pressure-meter --config=pressure_config.yaml
```

## 📦 安裝方式

### 預編譯版本下載

```bash
# Linux x64
wget https://github.com/yourusername/pressure-meter/releases/latest/download/pressure-meter-linux-amd64
chmod +x pressure-meter-linux-amd64
mv pressure-meter-linux-amd64 pressure-meter

# Linux ARM64
wget https://github.com/yourusername/pressure-meter/releases/latest/download/pressure-meter-linux-arm64

# Windows x64
# 下載 pressure-meter-windows-amd64.exe

# macOS
wget https://github.com/yourusername/pressure-meter/releases/latest/download/pressure-meter-darwin-amd64
```

### 從源碼編譯

```bash
# 克隆項目
git clone https://github.com/yourusername/pressure-meter.git
cd pressure-meter

# 安裝依賴
go mod download

# 編譯
go build -o pressure-meter

# 安裝到系統
sudo cp pressure-meter /usr/local/bin/
```

### Docker 安裝

```bash
# 拉取鏡像
docker pull yourusername/pressure-meter:latest

# 運行容器
docker run --privileged --device=/dev/ttyUSB0:/dev/ttyUSB0 \
  -e PRESSURE_DEVICE=/dev/ttyUSB0 \
  -e PRESSURE_SLAVE_ID=22 \
  yourusername/pressure-meter:latest
```

## 🎮 使用方法

### 基本命令

```bash
# 顯示幫助
./pressure-meter --help

# 顯示版本
./pressure-meter --version

# 自動掃描設備
./pressure-meter --auto-scan

# 快速掃描設備
./pressure-meter --quick-scan

# 完整掃描設備
./pressure-meter --full-scan

# 測試配置
./pressure-meter --test-config
```

### 高級用法

```bash
# JSON 格式輸出，運行 5 分鐘
./pressure-meter --output=json --duration=5m

# CSV 格式，最多 100 個讀數
./pressure-meter --output=csv --max-readings=100

# 詳細模式，保存日誌
./pressure-meter --verbose --log=pressure.log

# 守護程序模式
./pressure-meter --daemon --log=/var/log/pressure.log

# 指定配置檔案
./pressure-meter --config=my_config.yaml --interval=2s
```

### 輸出格式示例

#### 文本格式（默認）
```
[14:35:22] #1 站點22: 125.30 Pa (平均: 125.30 Pa)
[14:35:23] #2 站點22: 124.85 Pa (平均: 125.08 Pa)
[14:35:24] #3 站點22: 125.67 Pa (平均: 125.27 Pa)
```

#### JSON 格式
```json
{"timestamp":"2024-01-01T14:35:22Z","count":1,"slave_id":22,"pressure":125.30,"unit":"Pa","valid":true}
{"timestamp":"2024-01-01T14:35:23Z","count":2,"slave_id":22,"pressure":124.85,"unit":"Pa","valid":true}
```

#### CSV 格式
```csv
timestamp,count,slave_id,pressure,unit,valid
2024-01-01 14:35:22,1,22,125.300,Pa,true
2024-01-01 14:35:23,2,22,124.850,Pa,true
```

## ⚙️ 配置說明

### 環境變數配置

```bash
# 複製環境變數模板
cp .env.template .env

# 編輯配置
nano .env

# 載入環境變數
source .env
```

**主要環境變數：**

| 變數名 | 說明 | 示例值 | 默認值 |
|--------|------|--------|--------|
| `PRESSURE_DEVICE` | RS485 設備路徑 | `/dev/ttyUSB0` | `/dev/ttyUSB0` |
| `PRESSURE_SLAVE_ID` | Modbus 從站ID | `22` | `22` |
| `PRESSURE_DATA_FORMAT` | 數據格式 | `decimal` 或 `float` | `decimal` |
| `PRESSURE_READ_INTERVAL` | 讀取間隔 | `1s`, `500ms` | `1s` |
| `LOG_FILE` | 日誌檔案路徑 | `./logs/pressure.log` | - |
| `OUTPUT_FORMAT` | 輸出格式 | `text`, `json`, `csv` | `text` |

### 配置檔案格式

#### YAML 格式 (`pressure_config.yaml`)
```yaml
device: /dev/ttyUSB0
slaveid: 22
readinterval: 1s
dataformat: 0  # 0=十進制, 1=浮點數
```

#### JSON 格式 (`pressure_config.json`)
```json
{
  "device": "/dev/ttyUSB0",
  "slaveid": 22,
  "readinterval": "1s",
  "dataformat": 0
}
```

### 命令列參數

```bash
./pressure-meter \
  --device=/dev/ttyUSB0 \
  --slave-id=22 \
  --interval=1s \
  --format=decimal \
  --output=json \
  --log=pressure.log \
  --verbose
```

## 📊 API 文檔

### Go 套件使用

```go
package main

import (
    "log"
    "time"
    "pressure-meter/pressure"
)

func main() {
    // 創建配置
    config := pressure.Config{
        Device:       "/dev/ttyUSB0",
        SlaveID:      22,
        ReadInterval: time.Second,
        DataFormat:   pressure.DecimalFormat,
        Logger:       log.Default(),
    }

    // 創建設備實例
    pm, err := pressure.NewPressureMeter(config)
    if err != nil {
        log.Fatal(err)
    }
    defer pm.Close()

    // 開始監測
    pm.Start(config.ReadInterval)

    // 讀取數據
    for reading := range pm.GetReadings() {
        if reading.Valid {
            log.Printf("壓力: %.2f Pa", reading.Pressure)
        }
    }
}
```

### 自動掃描 API

```go
// 自動掃描並配置
scanner := pressure.NewScanner(log.Default())
config, err := scanner.AutoConfigure()
if err != nil {
    log.Fatal(err)
}

// 創建設備
pm, err := pressure.NewPressureMeter(*config)
```

### 壓力單位轉換

```go
// 創建測量值
measurement := pressure.Measurement{
    Value: 1500.0,
    Unit:  pressure.Pascal,
}

// 單位轉換
kpa := measurement.To(pressure.Kilopascal)  // 1.500 kPa
mbar := measurement.To(pressure.Millibar)   // 15.000 mbar
psi := measurement.To(pressure.PSI)         // 0.218 psi
```

## 🔧 故障排除

### 常見問題

#### 1. 無法找到設備
```bash
# 檢查設備是否存在
ls -la /dev/tty*

# 檢查權限
sudo usermod -a -G dialout $USER

# 重新登錄或重新載入組
newgrp dialout
```

#### 2. 權限被拒絕
```bash
# 檢查設備權限
ls -la /dev/ttyUSB0

# 添加用戶到 dialout 組
sudo usermod -a -G dialout $USER

# 或使用 sudo 運行
sudo ./pressure-meter --auto-scan
```

#### 3. 設備被占用
```bash
# 檢查哪個程序占用設備
lsof /dev/ttyUSB0

# 終止占用進程
sudo kill <PID>
```

#### 4. 讀取數據失敗
```bash
# 使用自動掃描檢測正確參數
./pressure-meter --auto-scan

# 測試配置
./pressure-meter --test-config

# 啟用詳細模式檢查錯誤
./pressure-meter --verbose
```

### 平台特定問題

#### Windows
```powershell
# 檢查 COM 端口
Get-WmiObject -Class Win32_SerialPort

# 使用設備管理器檢查驅動
devmgmt.msc
```

#### macOS
```bash
# 檢查串口設備
ls /dev/cu.*

# 安裝 Homebrew 和 Go (如果需要)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install go
```

#### Linux (樹莓派)
```bash
# 啟用串口
sudo raspi-config
# Interface Options > Serial Port > Enable

# 檢查串口配置
dmesg | grep tty
```

## 🔄 系統服務

### Systemd 服務 (Linux)

創建 `/etc/systemd/system/pressure-meter.service`：

```ini
[Unit]
Description=壓差儀監測服務
After=network.target

[Service]
Type=simple
User=pi
Group=dialout
WorkingDirectory=/home/pi/pressure-meter
Environment=PRESSURE_DEVICE=/dev/ttyUSB0
Environment=PRESSURE_SLAVE_ID=22
Environment=LOG_FILE=/var/log/pressure.log
ExecStart=/home/pi/pressure-meter/pressure-meter --daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
# 啟用服務
sudo systemctl enable pressure-meter.service
sudo systemctl start pressure-meter.service

# 檢查狀態
sudo systemctl status pressure-meter.service

# 查看日誌
sudo journalctl -u pressure-meter.service -f
```

### Windows 服務

使用 [NSSM](https://nssm.cc/) 創建 Windows 服務：

```cmd
# 下載並安裝 NSSM
nssm install PressureMeter

# 配置服務
Path: C:\pressure-meter\pressure-meter.exe
Arguments: --daemon --log=C:\logs\pressure.log
```

## 🛠️ 開發指南

### 項目結構

```
pressure-meter/
├── main.go                 # 主程式入口
├── pressure/               # 核心套件
│   ├── device.go          # 設備驅動
│   ├── config.go          # 配置管理
│   ├── scanner.go         # 自動掃描
│   └── types.go           # 數據類型
├── .env.template          # 環境變數模板
├── docker-compose.yml     # Docker Compose 配置
├── Dockerfile            # Docker 構建文件
├── go.mod               # Go 模組文件
├── go.sum               # 依賴校驗文件
└── README.md           # 項目說明
```

### 開發環境設置

```bash
# 克隆項目
git clone https://github.com/yourusername/pressure-meter.git
cd pressure-meter

# 安裝依賴
go mod download

# 運行測試
go test ./...

# 格式化代碼
go fmt ./...

# 靜態分析
go vet ./...
```

### 編譯選項

```bash
# 本地編譯
go build -o pressure-meter

# 交叉編譯
GOOS=linux GOARCH=amd64 go build -o pressure-meter-linux-amd64
GOOS=linux GOARCH=arm64 go build -o pressure-meter-linux-arm64
GOOS=windows GOARCH=amd64 go build -o pressure-meter-windows-amd64.exe
GOOS=darwin GOARCH=amd64 go build -o pressure-meter-darwin-amd64

# 優化編譯 (減小體積)
go build -ldflags="-s -w" -o pressure-meter

# 帶版本信息編譯
go build -ldflags="-X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o pressure-meter
```

### 測試

```bash
# 運行所有測試
go test ./...

# 運行特定包的測試
go test ./pressure

# 帶覆蓋率測試
go test -cover ./...

# 生成覆蓋率報告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📚 依賴套件

- **[goburrow/modbus](https://github.com/goburrow/modbus)** - Modbus 協議實現
- **[go.bug.st/serial](https://pkg.go.dev/go.bug.st/serial)** - 串口通信
- **[gopkg.in/yaml.v3](https://gopkg.in/yaml.v3)** - YAML 配置解析

## 🤝 貢獻指南

歡迎貢獻代碼！請按照以下步驟：

1. Fork 本項目
2. 創建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交修改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 創建 Pull Request

### 代碼規範

- 使用 `go fmt` 格式化代碼
- 添加適當的註釋和文檔
- 為新功能添加測試
- 遵循 Go 語言慣例


## 📈 更新日誌

### v1.0.2 (2025-06-23)
- ✨ 初始版本發布
- 🔍 支援自動設備掃描
- 📊 支援十進制和浮點數據格式
- 🌐 跨平台支援
---

**⭐ 如果這個項目對您有幫助，請給一個 Star！**
