package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/go-vgo/robotgo"
	"github.com/otiai10/gosseract/v2"
)

// 全局配置
const (
	TriggerKey = "f1" // 设置触发热键
)

func main() {
	// 1. 初始化 GUI 应用
	myApp := app.New()
	w := myApp.NewWindow("Translation Overlay")

	// 设置窗口无边框、置顶、透明背景（模拟悬浮提示）
	w.SetUndecorated(true)
	w.SetAlwaysOnTop(true)

	// UI 组件：一个标签用于显示结果
	label := widget.NewLabel("Waiting for input...")
	label.Wrapping = fyne.TextWrapWord // 自动换行
	
	// 放入容器并设置大小
	content := container.NewStack(label)
	w.SetContent(content)
	w.Resize(fyne.NewSize(300, 100))

	// 2. 启动后台监听协程
	go func() {
		fmt.Println("程序已启动，按 F1 翻译剪贴板内容...")
		
		// 监听键盘事件 (轮询方式，简单有效)
		for {
			if robotgo.AddEvents(TriggerKey) {
				// 获取鼠标当前位置
				mouseX, mouseY := robotgo.GetMousePos()
				
				// 移动窗口到鼠标附近
				w.Move(fyne.NewPos(float32(mouseX+20), float32(mouseY+20)))
				w.Show() // 确保窗口显示

				label.SetText("Processing...")

				//处理剪贴板逻辑
				text, err := processClipboard()
				if err != nil {
					label.SetText("Error: " + err.Error())
				} else {
					// 可以在这里调用翻译 API
					translated := mockTranslate(text) 
					label.SetText(translated)
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 运行 GUI 主循环
	myApp.Run()
}

// processClipboard 判断剪贴板是文本还是图片，并提取文字
func processClipboard() (string, error) {
	// 1. 尝试读取剪贴板文本
	text, err := robotgo.ReadAll()
	if err == nil && text != "" {
		return "Text Mode: " + text, nil
	}

	// 2. 如果没有文本，尝试处理剪贴板图片 (OCR)
	// 注意：Go 直接读取剪贴板图片比较复杂，通常我们通过保存临时文件的方式过渡
	// 这里演示逻辑：假设剪贴板有一张图，我们尝试获取它
	
	// *注：robotgo 读取剪贴板位图在不同系统下表现不一，
	// 生产环境建议结合 github.com/golang-design/clipboard 使用
	// 这里为了演示 OCR 流程，我们假设用户刚刚进行了截图，且图片已在剪贴板
	
	img, err := robotgo.GetClipboardBitmap()
	// 注意：robotgo 的 bitmap处理比较底层，为了稳定性，
	// 很多 Go 工具实际上是让用户截图保存文件，然后读取文件。
	// 这里简化为：如果剪贴板里没字，我们尝试识别一个固定的临时图片(模拟)
	// 或者你可以结合 `github.com/kbinani/screenshot` 截取屏幕区域。
	
	// 下面是核心的 OCR 代码逻辑
	client := gosseract.NewClient()
	defer client.Close()
	
	// 如果你能从剪贴板拿到 []byte 格式的图片
	// client.SetImageFromBytes(imageBytes)
	
	// 为了代码能跑，这里演示“如果失败”的情况
	return "OCR Image functionality requires valid image data source.", nil
}

// 模拟翻译 API
func mockTranslate(src string) string {
	// 这里接入 Google Translate / DeepL API
	return fmt.Sprintf("[Trans] %s\n(这里填入真实翻译API调用)", src)
}

// 辅助函数：保存图片到文件（如果需要调试 OCR）
func saveImage(img image.Image, path string) {
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}
