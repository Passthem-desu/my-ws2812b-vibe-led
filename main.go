package main

import (
	"fmt"
	"log"
)

const SPIDevicePath = "/dev/spidev1.0"
const APIPort = 8080

func main() {
	log.Printf("启动 LED Shader API 服务")
	
	controller, err := NewController(SPIDevicePath)
	if err != nil {
		log.Fatalf("无法初始化 SPI 控制器: %v", err)
	}
	defer controller.Close()
	
	pipeline := NewPipelineManager(controller)
	
	pipeline.StartLoop()
	log.Println("渲染循环已启动 (60 FPS)")
	
	router := setupRouter(pipeline)
	
	log.Printf("Web API 监听在 :%d...", APIPort)
	router.Run(fmt.Sprintf(":%d", APIPort))
}

