//go:build windows

package platform

import (
	"strings"
	"testing"
)

func TestTrayIPSummary(t *testing.T) {
	tests := []struct {
		name string
		ips  []string
		want string
	}{
		{
			name: "empty",
			want: "未检测到",
		},
		{
			name: "single",
			ips:  []string{"192.168.1.10"},
			want: "192.168.1.10",
		},
		{
			name: "multiple",
			ips:  []string{"192.168.1.10", "10.0.0.8", "172.16.0.4"},
			want: "192.168.1.10 等 3 个",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trayIPSummary(tt.ips); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestTrayTooltip(t *testing.T) {
	got := trayTooltip(true, true, []string{"192.168.1.10", "10.0.0.8"}, "0.0.0.0:1080", "0.0.0.0:8080")
	want := strings.Join([]string{
		"GoProxy 运行",
		"IP：192.168.1.10 等 2 个",
		"S5：0.0.0.0:1080",
		"HTTP：0.0.0.0:8080",
	}, "\n")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	got = trayTooltip(false, false, nil, "", "")
	if got != "GoProxy 停" {
		t.Fatalf("expected compact stopped tooltip, got %q", got)
	}
}
