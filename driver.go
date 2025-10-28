package main

/*
#cgo LDFLAGS: -ldriver
#include "c-driver/driver.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

const (
	LEDCount = C.LED_COUNT
)

type Controller struct {
	fd C.int
}

func NewController(device string) (*Controller, error) {
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	fd := C.ws2812_init(cDevice)
	if fd < 0 {
		return nil, fmt.Errorf("无法初始化 SPI 设备: fd < 0")
	}

	return &Controller{fd: fd}, nil
}

func (c *Controller) SendColors(colors []byte) error {
	if len(colors) != LEDCount*3 {
		return fmt.Errorf("颜色数组长度错误: 期望 %d 字节, 实际 %d 字节", LEDCount*3, len(colors))
	}

	ret := C.ws2812_send_colors(
		c.fd,
		(*C.uint8_t)(unsafe.Pointer(&colors[0])),
		C.int(LEDCount),
	)

	if ret < 0 {
		return fmt.Errorf("SPI 发送失败 (C 返回 %d)", ret)
	}

	return nil
}

func (c *Controller) Close() {
	C.ws2812_close(c.fd)
}

