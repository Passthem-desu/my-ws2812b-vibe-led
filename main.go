package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	spiDevice := flag.String("spi", "/dev/spidev1.0", "SPI 设备路径");
	apiPort := flag.Int("port", 8080, "Web API 监听端口");

	log.Printf("启动 LED Shader API 服务")
	
	controller, err := NewController(*spiDevice)
	if err != nil {
		log.Fatalf("无法初始化 SPI 控制器: %v", err)
	}
	defer controller.Close()
	
	pipeline := NewPipelineManager(controller)
	
	pipeline.StartLoop()
	log.Println("渲染循环已启动 (60 FPS)")
	
	router := setupRouter(pipeline)
	
	log.Printf("Web API 监听在 :%d...", *apiPort)
	router.Run(fmt.Sprintf(":%d", *apiPort))
}

