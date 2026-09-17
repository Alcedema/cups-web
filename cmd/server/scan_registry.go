package main

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"os"
	"sync"
	"time"

	"cups-web/internal/store"
)

// ── 扫描任务(issue #111) ─────────────────────────────────────────────────
//
// 与 driverJob 类似但独立管理:
//   - 允许并发(scanimage 不共享 apt/dpkg 全局锁,同一台扫描仪 SANE 后端会自行串行)
//   - 产物落 scanDir 磁盘,记录写 scan_records 表,前端可查历史
//   - 任务态(running/succeeded/failed/cancelled)保留在内存,过期后清理
//     内存态过期不删磁盘和数据库记录——那属于用户可见资产,由维护清理策略统一处理

const (
	scanJobRetention = time.Hour
	scanJobTimeout   = 5 * time.Minute

	scanStatusRunning   = "running"
	scanStatusSucceeded = "succeeded"
	scanStatusFailed    = "failed"
	scanStatusCancelled = "cancelled"
)

// scanBuffer 加锁封装,后台 goroutine 写入 stderr、handler 并发读取实时日志。
type scanBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *scanBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *scanBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// scanJob 的字段全部小写,访问统一由 scanJobsMu 保护;
// 对外序列化时先在锁内快照成 scanJobView,避免 JSON 编码读到撕裂状态。
type scanJob struct {
	id         string
	recordID   int64
	userID     int64
	device     string
	mode       string
	resolution int
	source     string
	format     string
	outputPath string
	filename   string
	status     string
	errMsg     string
	sizeBytes  int64
	startedAt  time.Time
	finishedAt time.Time
	logBuf     *scanBuffer
	cancel     context.CancelFunc
}

type scanJobView struct {
	ID         string `json:"id"`
	RecordID   int64  `json:"recordId,omitempty"`
	UserID     int64  `json:"-"` // 仅用于 handler 权限检查,不出前端
	Device     string `json:"device"`
	Mode       string `json:"mode"`
	Resolution int    `json:"resolution"`
	Source     string `json:"source,omitempty"`
	Format     string `json:"format"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	Log        string `json:"log,omitempty"`
	Error      string `json:"error,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

var (
	scanJobsMu sync.Mutex
	scanJobs   = map[string]*scanJob{}
	scanDir    string
)

// viewLocked 必须在持有 scanJobsMu 时调用。
func (j *scanJob) viewLocked() *scanJobView {
	v := &scanJobView{
		ID:         j.id,
		RecordID:   j.recordID,
		UserID:     j.userID,
		Device:     j.device,
		Mode:       j.mode,
		Resolution: j.resolution,
		Source:     j.source,
		Format:     j.format,
		Filename:   j.filename,
		Status:     j.status,
		Log:        j.logBuf.String(),
		Error:      j.errMsg,
		SizeBytes:  j.sizeBytes,
		StartedAt:  j.startedAt.Format(time.RFC3339),
	}
	if !j.finishedAt.IsZero() {
		v.FinishedAt = j.finishedAt.Format(time.RFC3339)
	}
	return v
}

// pruneScanJobsLocked 清理完成时间超过 scanJobRetention 的内存态任务(不动磁盘/DB)。
func pruneScanJobsLocked() {
	cutoff := time.Now().Add(-scanJobRetention)
	for id, j := range scanJobs {
		if j.status != scanStatusRunning && !j.finishedAt.IsZero() && j.finishedAt.Before(cutoff) {
			delete(scanJobs, id)
		}
	}
}

// registerScanJob 登记新任务并启动后台 goroutine。
// runFn 内实际调用 scanimage + 后处理,返回产物大小(字节)和错误。
func registerScanJob(recordID, userID int64, device, mode string, resolution int, source, format, outputPath, filename string,
	runFn func(ctx context.Context, job *scanJob) (int64, error)) *scanJob {
	scanJobsMu.Lock()
	pruneScanJobsLocked()

	job := &scanJob{
		id:         randomToken(),
		recordID:   recordID,
		userID:     userID,
		device:     device,
		mode:       mode,
		resolution: resolution,
		source:     source,
		format:     format,
		outputPath: outputPath,
		filename:   filename,
		status:     scanStatusRunning,
		startedAt:  time.Now(),
		logBuf:     &scanBuffer{},
	}
	scanJobs[job.id] = job
	scanJobsMu.Unlock()

	// context.Background() 派生,避免 http.Server WriteTimeout 掐断子进程,
	// 与 driver_handlers.go 的做法保持一致。
	ctx, cancel := context.WithTimeout(context.Background(), scanJobTimeout)
	scanJobsMu.Lock()
	job.cancel = cancel
	scanJobsMu.Unlock()

	go func() {
		defer cancel()
		size, runErr := runFn(ctx, job)

		scanJobsMu.Lock()
		job.finishedAt = time.Now()
		job.sizeBytes = size
		switch {
		case runErr == nil:
			job.status = scanStatusSucceeded
		case ctx.Err() == context.Canceled && job.status == scanStatusCancelled:
			// 已由 handler 主动取消,保留 cancelled 状态
			job.errMsg = "任务已取消"
		default:
			job.status = scanStatusFailed
			job.errMsg = runErr.Error()
		}
		finalStatus := job.status
		finalErr := job.errMsg
		scanJobsMu.Unlock()

		if runErr != nil {
			log.Printf("[scan-job] %s 失败 (device=%s): %v", job.id, job.device, runErr)
			// 失败时清理磁盘产物(可能是半成品),数据库记录保留但 status=failed
			_ = os.Remove(job.outputPath)
			size = 0
		}

		// 写回数据库最终状态
		if recordID > 0 {
			finishedAt := job.finishedAt.UTC().Format(time.RFC3339)
			if err := appStore.WithTx(context.Background(), false, func(tx *sql.Tx) error {
				return store.UpdateScanStatus(context.Background(), tx, recordID, finalStatus, finalErr, size, finishedAt)
			}); err != nil {
				log.Printf("[scan-job] %s 更新记录失败: %v", job.id, err)
			}
		}
	}()

	return job
}

// findScanJob 按 id 取任务视图(锁内快照)。
func findScanJob(id string) *scanJobView {
	scanJobsMu.Lock()
	defer scanJobsMu.Unlock()
	if job, ok := scanJobs[id]; ok {
		return job.viewLocked()
	}
	return nil
}

// cancelScanJob 请求取消正在跑的任务;非 running 状态返回 false。
func cancelScanJob(id string) bool {
	scanJobsMu.Lock()
	defer scanJobsMu.Unlock()
	job, ok := scanJobs[id]
	if !ok || job.status != scanStatusRunning {
		return false
	}
	job.status = scanStatusCancelled
	if job.cancel != nil {
		job.cancel()
	}
	return true
}
