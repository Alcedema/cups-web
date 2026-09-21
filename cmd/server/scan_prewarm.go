package main

import (
	"context"
	"log"
	"time"
)

// startScanPrewarm 启动后一次性预热扫描设备缓存(issue #111 复测反馈)。
// scanimage -L 本身要 ~17s,不在这里跑就是让第一位打开扫描页的用户等这段时间。
// 缓存被禁用时(SCAN_DEVICES_CACHE_TTL<=0)直接跳过,保持旧行为不变。
func startScanPrewarm() {
	if scanDeviceCacheDisabled() {
		return
	}
	go func() {
		// 稍等 http 端口先监听、日志更清晰;不阻塞 main。
		time.Sleep(2 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		devices, _, err := scanDeviceCacheGet(ctx, true)
		if err != nil {
			log.Printf("[scan] 预热失败(不影响启动): %v", err)
			return
		}
		log.Printf("[scan] 预热完成,发现 %d 台扫描设备", len(devices))
	}()
}
