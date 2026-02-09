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
	"github.com/otiai10/gosseract/v2"
)

// --- 配置结构 (对应你的 JSON) ---
type AppConfig struct {
	AppID    string `json:"app_id"`
	AppKey   string `json:"app_key"`
	FromLang string `json:"from_lang"`
	ToLang   string `json:"to_lang"`
	Cuid     string `json:"cuid"`
	Mac      string `json:"mac"`
	CopyMode string `json:"copy_mode"`
}

// --- 百度 API 响应 ---
type BaiduResponse struct {
	TransResult []struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	} `json:"trans_result"`
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

var config AppConfig
const ConfigFile = "config.json"
const TriggerKey = "f1"

func main() {
	// 1. 加载配置
	if err := loadConfig(); err != nil {
		fmt.Println("Error: config.json not found or invalid.")
		// 如果没有配置，生成一个默认的防止崩溃
		config = AppConfig{FromLang: "auto", ToLang: "zh"}
	}

	// 2. GUI 初始化
	myApp := app.New()
	w := myApp.NewWindow("Translator")
	w.SetUndecorated(true)
	w.SetAlwaysOnTop(true)

	label := widget.NewLabel("Ready. Press " + TriggerKey)
	label.Wrapping = fyne.TextWrapWord
	
	// 稍微美化一下背景
	bg := container.NewMax(widget.NewCard("", "", label)) 
	w.SetContent(bg)
	w.Resize(fyne.NewSize(300, 100))

	// 3. 监听循环
	go func() {
		for {
			if robotgo.AddEvents(TriggerKey) {
				// 获取鼠标位置，移动窗口
				mx, my := robotgo.GetMousePos()
				w.Move(fyne.NewPos(float32(mx+20), float32(my+20)))
				w.Show()
				label.SetText("Analyzing...")

				// 尝试处理逻辑
				go func() {
					text := getContentFromClipboardOrOCR()
					if text == "" {
						label.SetText("No text found.")
						return
					}
					
					translated := baiduTranslate(text)
					label.SetText(translated)

					// 复制结果逻辑
					if config.CopyMode == "dst" {
						robotgo.WriteAll(translated)
					}
				}()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	myApp.Run()
}

// --- 核心逻辑：获取内容 ---
func getContentFromClipboardOrOCR() string {
	// 1. 尝试直接获取文本
	text, err := robotgo.ReadAll()
	if err == nil && strings.TrimSpace(text) != "" {
		return text
	}

	// 2. 如果文本为空，尝试获取剪贴板图片进行 OCR
	// Robotgo 获取图片并不总稳定，这里尝试用 Gosseract 从剪贴板读取
	// 注意：在 Windows 上，Go 直接读取剪贴板图片比较复杂
	// 为了演示，这里假设用户已经把截图保存在了剪贴板，我们尝试通过 OCR 客户端读取
	// 实际生产中通常结合 github.com/kbinani/screenshot 截图
	
	client := gosseract.NewClient()
	defer client.Close()
	
	// 这里使用 Gosseract 的剪贴板功能（依赖系统库）
	// 如果无法直接从内存读，通常策略是：Save clipboard to temp file -> OCR temp file
	// 这里简化处理，尝试直接从剪贴板获取文本（部分 Tesseract 版本支持）
	// 或者您可以添加截图功能：
	// robotgo.SaveCapture("temp.png") -> client.SetImage("temp.png")
	
	return "" // 暂时返回空，因为纯剪贴板图片 OCR 代码较长，需要额外库
}

// --- 百度翻译 ---
func baiduTranslate(q string) string {
	if config.AppID == "" {
		return "Error: AppID not configured"
	}

	salt := strconv.Itoa(rand.Intn(100000))
	// 签名 = MD5(appid + q + salt + key)
	sign := md5Hash(config.AppID + q + salt + config.AppKey)

	// 构建 URL
	u, _ := url.Parse("http://api.fanyi.baidu.com/api/trans/vip/translate")
	params := url.Values{}
	params.Set("q", q)
	params.Set("from", config.FromLang)
	params.Set("to", config.ToLang)
	params.Set("appid", config.AppID)
	params.Set("salt", salt)
	params.Set("sign", sign)
	u.RawQuery = params.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return "Net Error"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res BaiduResponse
	json.Unmarshal(body, &res)

	if res.ErrorCode != "" && res.ErrorCode != "52000" {
		return "API Error: " + res.ErrorCode
	}

	var sb strings.Builder
	for _, r := range res.TransResult {
		sb.WriteString(r.Dst + "\n")
	}
	return sb.String()
}

func loadConfig() error {
	f, err := os.Open(ConfigFile)
	if err != nil { return err }
	defer f.Close()
	return json.NewDecoder(f).Decode(&config)
}

func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
