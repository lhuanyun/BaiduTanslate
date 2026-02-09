package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/go-vgo/robotgo"
	// "github.com/otiai10/gosseract/v2" // 如果暂时不用OCR，可以注释掉以加快编译速度
)

// ---------------------- 配置结构体 ----------------------
type AppConfig struct {
	AppID    string `json:"app_id"`
	AppKey   string `json:"app_key"`
	FromLang string `json:"from_lang"`
	ToLang   string `json:"to_lang"`
	CopyMode string `json:"copy_mode"` // src: 不动, dst: 复制翻译结果
}

// ---------------------- 百度 API 响应结构体 ----------------------
type BaiduResponse struct {
	TransResult []struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	} `json:"trans_result"`
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

// 全局变量
var config AppConfig
const ConfigFile = "config.json"
const TriggerKey = "f1"

func main() {
	// 1. 加载配置
	if err := loadConfig(); err != nil {
		fmt.Println("无法读取 config.json:", err)
		fmt.Println("请确保 config.json 在程序同一目录下")
		return // 配置读取失败直接退出
	}
	fmt.Printf("配置已加载: AppID=%s, From=%s, To=%s\n", config.AppID, config.FromLang, config.ToLang)

	// 2. 初始化 GUI
	myApp := app.New()
	w := myApp.NewWindow("Translator")
	w.SetUndecorated(true)
	w.SetAlwaysOnTop(true)

	// 结果显示标签
	label := widget.NewLabel("Ready. Press " + TriggerKey)
	label.Wrapping = fyne.TextWrapWord
	
	// 背景容器 (加一点内边距)
	content := container.NewStack(label)
	w.SetContent(content)
	w.Resize(fyne.NewSize(300, 100))

	// 3. 后台监听协程
	go func() {
		for {
			if robotgo.AddEvents(TriggerKey) {
				// 获取鼠标位置并移动窗口
				mouseX, mouseY := robotgo.GetMousePos()
				w.Move(fyne.NewPos(float32(mouseX+20), float32(mouseY+20)))
				w.Show()
				label.SetText("Translating...")

				// 获取剪贴板文本
				text, err := robotgo.ReadAll()
				if err != nil || text == "" {
					label.SetText("Clipboard is empty or image (OCR not enabled)")
					// 如果需要 OCR，在这里调用之前的 gosseract 逻辑
				} else {
					// 调用百度翻译
					result := baiduTranslate(text)
					label.SetText(result)

					// 根据配置处理剪贴板 (如果 copy_mode 是 dst，则将结果写入剪贴板)
					if config.CopyMode == "dst" {
						robotgo.WriteAll(result)
					}
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	myApp.Run()
}

// ---------------------- 核心逻辑 ----------------------

// 读取配置文件
func loadConfig() error {
	file, err := os.Open(ConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	return decoder.Decode(&config)
}

// 百度翻译实现
func baiduTranslate(query string) string {
	apiURL := "http://api.fanyi.baidu.com/api/trans/vip/translate"
	
	// 生成随机盐值
	salt := strconv.Itoa(rand.Intn(100000))

	// 生成签名: MD5(appid + q + salt + key)
	signStr := config.AppID + query + salt + config.AppKey
	sign := md5Hash(signStr)

	// 构建请求参数
	params := url.Values{}
	params.Set("q", query)
	params.Set("from", config.FromLang)
	params.Set("to", config.ToLang)
	params.Set("appid", config.AppID)
	params.Set("salt", salt)
	params.Set("sign", sign)

	// 发送 POST 请求
	resp, err := http.PostForm(apiURL, params)
	if err != nil {
		return "Network Error: " + err.Error()
	}
	defer resp.Body.Close()

	// 读取响应
	body, _ := io.ReadAll(resp.Body)
	
	// 解析 JSON
	var baiduResp BaiduResponse
	if err := json.Unmarshal(body, &baiduResp); err != nil {
		return "JSON Error"
	}

	// 错误处理
	if baiduResp.ErrorCode != "" && baiduResp.ErrorCode != "52000" {
		return fmt.Sprintf("API Error: %s - %s", baiduResp.ErrorCode, baiduResp.ErrorMsg)
	}

	// 拼接结果 (可能有多段)
	var finalResult strings.Builder
	for _, res := range baiduResp.TransResult {
		finalResult.WriteString(res.Dst)
		finalResult.WriteString("\n")
	}

	return strings.TrimSpace(finalResult.String())
}

// MD5 工具函数
func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
