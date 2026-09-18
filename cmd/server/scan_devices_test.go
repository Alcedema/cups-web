package main

import "testing"

// 覆盖 escl (eSCL/AirScan) 后端的 `scanimage -A` 输出:关键在于同时列出
// `--br-x`/`--br-y` 的 range 上限,让 runScanimage 能显式补几何(issue #112)。
const scanOptionsEsclSample = `
All options specific to device ` + "`escl:https://192.0.2.10:443`" + `:
  Scan mode:
    --mode Color|Gray|Lineart [Color]
        Selects the scan mode (e.g., lineart, monochrome, or color).
    --resolution 75|100|150|200|300|600dpi [150]
        Sets the resolution of the scanned image.
    --source Flatbed|ADF [Flatbed]
        Selects the scan source (such as a document-feeder).
  Geometry:
    -l 0..215.9mm (in steps of 0.0211639) [0]
        Top-left x position of scan area.
    -t 0..297mm (in steps of 0.0211639) [0]
        Top-left y position of scan area.
    --tl-x 0..215.9mm (in steps of 0.0211639) [0]
        Top-left x position of scan area.
    --tl-y 0..297mm (in steps of 0.0211639) [0]
        Top-left y position of scan area.
    --br-x 0..215.9mm (in steps of 0.0211639) [215.9]
        Bottom-right x position of scan area.
    --br-y 0..297mm (in steps of 0.0211639) [297]
        Bottom-right y position of scan area.
`

// hpaio 后端(HP LaserJet M1005)的样例,没有 br-x/br-y,几何用 -x/-y。
// runScanimage 走 -x/-y 分支;不追加原点。
const scanOptionsHpaioSample = `
All options specific to device ` + "`hpaio:/usb/HP_LaserJet_M1005`" + `:
  Scan mode:
    --mode Lineart|Gray|Color [Gray]
    --resolution 75|100|150|200|300|600dpi [75]
    --source Flatbed [Flatbed]
  Geometry:
    -l 0..215mm [0]
    -t 0..297mm [0]
    -x 0..215mm [215]
    -y 0..297mm [297]
`

// 老驱动只暴露 mode/resolution,没有任何几何选项 → 保持"不追加"。
const scanOptionsMinimalSample = `
  Scan mode:
    --mode Color|Gray [Gray]
    --resolution 150|300 [150]
`

func TestParseScanOptions_EsclHasBrGeometry(t *testing.T) {
	opts := parseScanOptions(scanOptionsEsclSample)

	brX, ok := opts["br-x"]
	if !ok {
		t.Fatal("escl 样例应解析出 br-x")
	}
	if brX.Type != "range" {
		t.Fatalf("br-x.Type = %q, want range", brX.Type)
	}
	if brX.Max != "215.9" {
		t.Fatalf("br-x.Max = %q, want 215.9", brX.Max)
	}

	brY, ok := opts["br-y"]
	if !ok {
		t.Fatal("escl 样例应解析出 br-y")
	}
	if brY.Max != "297" {
		t.Fatalf("br-y.Max = %q, want 297", brY.Max)
	}

	// mode / resolution / source 与既有行为保持一致
	if opts["mode"].Type != "list" || len(opts["mode"].Values) != 3 {
		t.Fatalf("mode = %+v, want list of 3", opts["mode"])
	}
	if opts["source"].Type != "list" {
		t.Fatalf("source.Type = %q, want list", opts["source"].Type)
	}
}

// 短开关 -l/-t 出现在 --tl-x/--tl-y 之前时,不覆盖已存在的长开关条目。
// (parseScanOptions 内部按行扫,先出现的短开关会先落 tl-x,长开关的 range 应写进 map。)
// 用一份"只有短开关"的样例反过来验证:短开关也能填 tl-x/tl-y。
func TestParseScanOptions_ShortSwitchesFillTLKeys(t *testing.T) {
	raw := `
  Geometry:
    -l 0..210mm (in steps of 0.1) [0]
    -t 0..297mm (in steps of 0.1) [0]
`
	opts := parseScanOptions(raw)
	if opts["tl-x"].Max != "210" {
		t.Fatalf("tl-x.Max = %q, want 210", opts["tl-x"].Max)
	}
	if opts["tl-y"].Max != "297" {
		t.Fatalf("tl-y.Max = %q, want 297", opts["tl-y"].Max)
	}
}

func TestParseScanOptions_HpaioHasXYNotBr(t *testing.T) {
	opts := parseScanOptions(scanOptionsHpaioSample)

	if _, ok := opts["br-x"]; ok {
		t.Fatal("hpaio 样例不应解析出 br-x")
	}
	if _, ok := opts["br-y"]; ok {
		t.Fatal("hpaio 样例不应解析出 br-y")
	}
	if opts["x"].Max != "215" {
		t.Fatalf("x.Max = %q, want 215", opts["x"].Max)
	}
	if opts["y"].Max != "297" {
		t.Fatalf("y.Max = %q, want 297", opts["y"].Max)
	}
}

func TestParseScanOptions_MinimalNoGeometry(t *testing.T) {
	opts := parseScanOptions(scanOptionsMinimalSample)
	for _, k := range []string{"br-x", "br-y", "tl-x", "tl-y", "x", "y"} {
		if _, ok := opts[k]; ok {
			t.Fatalf("minimal 样例不应包含几何键 %q", k)
		}
	}
	if opts["mode"].Type != "list" {
		t.Fatalf("mode.Type = %q, want list", opts["mode"].Type)
	}
}

func TestIsNumericScanValue(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"0", true},
		{"215.9", true},
		{"-1", true},
		{"-3.14", true},
		{"", false},
		{"215.9mm", false},            // 带单位,拒绝
		{"215.9 --format=pdf", false}, // 试图注入其它 flag
		{"215.9;rm -rf /", false},     // 试图注入 shell(即使 exec 无 shell,也拒绝)
		{"1e3", false},                // 科学计数法不接受
		{".5", false},                 // 必须有整数部分
	}
	for _, c := range cases {
		if got := isNumericScanValue(c.in); got != c.want {
			t.Errorf("isNumericScanValue(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
