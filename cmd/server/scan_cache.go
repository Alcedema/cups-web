package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"
)

// 背景(issue #111 复测反馈):每次进扫描页调 `scanimage -L` 需 ~17s
// (多种 SANE 后端串行发现累加而来)。这里给设备发现结果加一层进程内
// TTL 缓存,避免频繁切页/多接口共同触发的重复等待:
//
//   - `GET /api/scan/devices`  默认走缓存;`?force=1` 打穿并刷新
//   - probe / options / jobs 校验设备存在的三处调用同样过缓存,把设备列表
//     当作近实时字典使用,不承担"发现设备"职责
//
// 缓存策略:
//   - TTL 由 SCAN_DEVICES_CACHE_TTL 覆盖(Go time.ParseDuration),默认 60s
//   - <= 0 视为禁用,回退到旧行为(每次真调 scanimage -L)
//   - 失败不覆盖已有 entry:避免"点了刷新失败反而丢掉热数据"
//   - 手写单飞:多请求撞同一次刷新时只发一次子进程,其余 goroutine 等结果

const defaultScanDeviceCacheTTL = 60 * time.Second

type scanDeviceCacheEntry struct {
	devices []ScanDevice
	raw     string
	fetched time.Time
}

type scanDeviceLoad struct {
	done    chan struct{}
	devices []ScanDevice
	raw     string
	err     error
}

type scanDeviceCache struct {
	mu       sync.Mutex
	ttl      time.Duration
	inflight *scanDeviceLoad
	entry    *scanDeviceCacheEntry
}

var defaultScanDeviceCache = newScanDeviceCacheFromEnv()

// scanDeviceCacheGet 从缓存拿设备列表,过期或 force=true 时触发一次真调。
// 与 listScanDevices 保持同样的三返回值,便于 handler 无侵入替换。
func scanDeviceCacheGet(ctx context.Context, force bool) ([]ScanDevice, string, error) {
	return defaultScanDeviceCache.get(ctx, force)
}

// scanDeviceCacheDisabled 供预热逻辑判断是否需要跑。
func scanDeviceCacheDisabled() bool {
	return defaultScanDeviceCache.ttl <= 0
}

func newScanDeviceCacheFromEnv() *scanDeviceCache {
	ttl := defaultScanDeviceCacheTTL
	if v := os.Getenv("SCAN_DEVICES_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		} else {
			log.Printf("[scan] 无法解析 SCAN_DEVICES_CACHE_TTL=%q: %v,回退到 %s", v, err, defaultScanDeviceCacheTTL)
		}
	}
	return &scanDeviceCache{ttl: ttl}
}

func (c *scanDeviceCache) get(ctx context.Context, force bool) ([]ScanDevice, string, error) {
	if c.ttl <= 0 {
		return listScanDevices(ctx)
	}

	c.mu.Lock()
	if !force && c.entry != nil && time.Since(c.entry.fetched) < c.ttl {
		e := c.entry
		c.mu.Unlock()
		return e.devices, e.raw, nil
	}
	if c.inflight != nil {
		load := c.inflight
		c.mu.Unlock()
		return waitScanDeviceLoad(ctx, load)
	}
	load := &scanDeviceLoad{done: make(chan struct{})}
	c.inflight = load
	c.mu.Unlock()

	// 单飞:只有拿到 inflight 位置的 goroutine 真的调 listScanDevices。
	// 用独立的 context.Background 派生超时,避免调用方 ctx 提前取消把
	// 结果扔给其它 waiter——它们的 ctx 可能还长着(比如 probe 有独立超时)。
	loadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	go func() {
		defer cancel()
		devices, raw, err := listScanDevices(loadCtx)
		c.mu.Lock()
		if err == nil {
			c.entry = &scanDeviceCacheEntry{
				devices: devices,
				raw:     raw,
				fetched: time.Now(),
			}
		}
		c.inflight = nil
		c.mu.Unlock()
		load.devices = devices
		load.raw = raw
		load.err = err
		close(load.done)
	}()

	return waitScanDeviceLoad(ctx, load)
}

func waitScanDeviceLoad(ctx context.Context, load *scanDeviceLoad) ([]ScanDevice, string, error) {
	select {
	case <-load.done:
		return load.devices, load.raw, load.err
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
}
