package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ScanDevice 是 `scanimage -L` 解析出的一台设备(issue #111)。
type ScanDevice struct {
	// Name 是 SANE 设备 URI(如 hpaio:/usb/HP_LaserJet_M1005?serial=XXX 或
	// hpljm1005:libusb:001:002),提交扫描任务时按名字直接传给 scanimage -d。
	Name   string `json:"name"`
	Vendor string `json:"vendor,omitempty"`
	Model  string `json:"model,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

// ScanOption 描述某台设备一项参数的可选值。value 类型统一为字符串,
// 前端拿到后决定渲染成下拉框还是数字输入。
type ScanOption struct {
	Name    string   `json:"name"`
	Title   string   `json:"title,omitempty"`
	Type    string   `json:"type"` // list / range / string
	Values  []string `json:"values,omitempty"`
	Min     string   `json:"min,omitempty"`
	Max     string   `json:"max,omitempty"`
	Step    string   `json:"step,omitempty"`
	Unit    string   `json:"unit,omitempty"`
	Default string   `json:"default,omitempty"`
}

// listScanDevices 调用 scanimage -L 列出可用设备。
//
// scanimage -L 输出格式(每行一台):
//
//	device `hpaio:/usb/HP_LaserJet_M1005?serial=XXX' is a Hewlett-Packard HP_LaserJet_M1005 all-in-one
//	device `hpljm1005:libusb:001:002' is a Hewlett-Packard hpljm1005 multi-function peripheral
//
// 特别行 "No scanners were identified." 表示零设备,直接返回空切片。
func listScanDevices(ctx context.Context) ([]ScanDevice, string, error) {
	// -q 只输出设备行,不输出"searching..." 之类的进度提示。
	// 部分 SANE 版本 -q 与 -L 组合仍会打印发现进度到 stderr,不影响 stdout 解析。
	cmd := exec.CommandContext(ctx, "scanimage", "-L", "-q")
	out, err := cmd.CombinedOutput()
	raw := string(out)
	if err != nil {
		return nil, raw, fmt.Errorf("scanimage -L 失败: %w", err)
	}

	devices := []ScanDevice{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(strings.ToLower(line), "no scanners were identified") {
			return devices, raw, nil
		}
		dev, ok := parseScanDeviceLine(line)
		if ok {
			devices = append(devices, dev)
		}
	}
	return devices, raw, nil
}

var scanDeviceLineRegexp = regexp.MustCompile("^device `([^']+)' is a (.+)$")

func parseScanDeviceLine(line string) (ScanDevice, bool) {
	m := scanDeviceLineRegexp.FindStringSubmatch(line)
	if len(m) != 3 {
		return ScanDevice{}, false
	}
	name := strings.TrimSpace(m[1])
	desc := strings.TrimSpace(m[2])
	dev := ScanDevice{Name: name}
	// desc 通常形如 "Hewlett-Packard HP_LaserJet_M1005 all-in-one",
	// 按空格拆一次,首段当 vendor,末段当 kind,中间当 model。
	parts := strings.Fields(desc)
	switch {
	case len(parts) >= 3:
		dev.Vendor = parts[0]
		dev.Kind = parts[len(parts)-1]
		dev.Model = strings.Join(parts[1:len(parts)-1], " ")
	case len(parts) == 2:
		dev.Vendor = parts[0]
		dev.Model = parts[1]
	case len(parts) == 1:
		dev.Model = parts[0]
	}
	return dev, true
}

// listScanOptions 调用 `scanimage -A -d <device>` 列出该设备的可选参数
// (issue #111 / #112)。提取的字段见 parseScanOptions。
//
// 输出格式片段(SANE 官方 CLI):
//
//	-x 0..215.9mm (in steps of 0.0211639) [215.9]
//	--mode Color|Gray|Lineart [Gray]
//	--resolution 75|100|150|200|300|600dpi [150]
//	--source ADF|Flatbed [Flatbed]
//	--br-x 0..215.9mm (in steps of 0.0211639) [215.9]
//
// 部分驱动会给出 range 而不是枚举(--resolution 50..600dpi (in steps of 1))。
// 解析逻辑对两种格式都容忍。
func listScanOptions(ctx context.Context, device string) (map[string]ScanOption, string, error) {
	cmd := exec.CommandContext(ctx, "scanimage", "-A", "-d", device)
	out, err := cmd.CombinedOutput()
	raw := string(out)
	if err != nil {
		return nil, raw, fmt.Errorf("scanimage -A -d %q 失败: %w", device, err)
	}
	opts := parseScanOptions(raw)
	return opts, raw, nil
}

// parseScanOptions 解析 `scanimage -A` 的输出,提取 MVP 需要的参数。
//
//   - mode / resolution / source: 前端表单直接使用
//   - br-x / br-y / tl-x / tl-y: 用于给 escl (eSCL/AirScan) 后端显式指定扫描区域,
//     否则默认 br-x/br-y 会被 rounded to 0,触发 sane_start: Invalid argument
//     (issue #112)
//   - 短开关 -l/-t/-x/-y: 老式 SANE 后端语义;-l/-t 对齐到 tl-x/tl-y,
//     -x/-y 是窗口宽/高,单独保留为 "x"/"y"
//
// 兼容以 `--<name>` 或 `-<x>` 开头的行;默认值(方括号)可选,缺省时留空。
func parseScanOptions(raw string) map[string]ScanOption {
	opts := map[string]ScanOption{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-") {
			continue
		}
		nameEnd := strings.IndexAny(line, " \t")
		if nameEnd < 0 {
			continue
		}
		head := line[:nameEnd]
		body := strings.TrimSpace(line[nameEnd:])

		var name string
		switch {
		case strings.HasPrefix(head, "--"):
			name = strings.TrimPrefix(head, "--")
		case len(head) == 2: // 短开关 "-l" / "-x" 等
			switch head[1] {
			case 'l':
				name = "tl-x"
			case 't':
				name = "tl-y"
			case 'x':
				name = "x"
			case 'y':
				name = "y"
			default:
				continue
			}
		default:
			continue
		}

		switch name {
		case "mode", "resolution", "source",
			"br-x", "br-y", "tl-x", "tl-y", "x", "y":
			// 短开关和长开关同名时,不覆盖已有条目——SANE 通常同时列出
			// "-l ..." 和后续的 "--tl-x ..." (若存在),二者语义等价。
			if _, ok := opts[name]; !ok {
				opts[name] = parseScanOptionBody(name, body)
			}
		}
	}
	return opts
}

// scanNumericValueRegexp 用于校验 parseScanOptions 抽出的 min/max/default,
// 只允许可选负号 + 数字 + 可选小数部分。runScanimage 会把这些值再拼回给
// scanimage 子进程,守卫住此正则可以避免 -A 输出被污染后注入其它参数。
var scanNumericValueRegexp = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// isNumericScanValue 判断 s 是否为合法的数字型 scanimage 参数值。
func isNumericScanValue(s string) bool {
	return scanNumericValueRegexp.MatchString(s)
}

var scanOptionRangeRegexp = regexp.MustCompile(`^([\-\d.]+)\.\.([\-\d.]+)([a-zA-Z]*)`)
var scanOptionStepRegexp = regexp.MustCompile(`in steps of ([\-\d.]+)`)
var scanOptionDefaultRegexp = regexp.MustCompile(`\[([^\]]*)\]\s*$`)

func parseScanOptionBody(name, body string) ScanOption {
	opt := ScanOption{Name: name, Title: name}
	// 抽默认值(方括号在末尾)
	if m := scanOptionDefaultRegexp.FindStringSubmatch(body); len(m) == 2 {
		opt.Default = strings.TrimSpace(m[1])
		body = strings.TrimSpace(body[:len(body)-len(m[0])])
	}

	// range 形式: "0..300dpi (in steps of 1)"
	if m := scanOptionRangeRegexp.FindStringSubmatch(body); len(m) >= 3 {
		opt.Type = "range"
		opt.Min = m[1]
		opt.Max = m[2]
		if len(m) >= 4 {
			opt.Unit = m[3]
		}
		if sm := scanOptionStepRegexp.FindStringSubmatch(body); len(sm) == 2 {
			opt.Step = sm[1]
		}
		return opt
	}

	// 枚举形式: "Color|Gray|Lineart" 或 "75|150|300dpi"
	if strings.Contains(body, "|") {
		// 去掉末尾单位(如 "dpi")
		trimmed := body
		if idx := strings.IndexAny(trimmed, " ("); idx >= 0 {
			trimmed = trimmed[:idx]
		}
		vals := strings.Split(trimmed, "|")
		// 从最后一个值里剥单位后缀(如 "600dpi" → 600 + dpi)
		if len(vals) > 0 {
			last := vals[len(vals)-1]
			if idx := indexOfSuffixLetters(last); idx > 0 {
				opt.Unit = last[idx:]
				vals[len(vals)-1] = last[:idx]
			}
		}
		for i := range vals {
			vals[i] = strings.TrimSpace(vals[i])
		}
		opt.Type = "list"
		opt.Values = vals
		return opt
	}

	opt.Type = "string"
	return opt
}

// indexOfSuffixLetters 从字符串末尾往前找,返回第一个纯字母后缀的起始下标;
// 例如 "600dpi" 返回 3,"Flatbed" 返回 0。
func indexOfSuffixLetters(s string) int {
	i := len(s)
	for i > 0 {
		r := s[i-1]
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			i--
			continue
		}
		break
	}
	return i
}

// parseResolution 只是薄封装,给 handler 用来把用户传的分辨率字符串转 int
// 后再作为 scanimage 参数拼回去(防注入的核心手段)。
func parseResolution(s string) (int, error) {
	// 兼容 "300dpi" 这种带单位的输入
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimSuffix(s, "dpi")
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < 50 || n > 4800 {
		return 0, fmt.Errorf("resolution %d 超出 50..4800 范围", n)
	}
	return n, nil
}
