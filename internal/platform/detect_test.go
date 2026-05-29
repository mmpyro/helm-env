package platform

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	info, err := Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		if info.OS != "darwin" {
			t.Fatalf("expected darwin, got %s", info.OS)
		}
	case "linux":
		if info.OS != "linux" {
			t.Fatalf("expected linux, got %s", info.OS)
		}
	}

	switch runtime.GOARCH {
	case "amd64":
		if info.Arch != "amd64" {
			t.Fatalf("expected amd64, got %s", info.Arch)
		}
	case "arm64":
		if info.Arch != "arm64" {
			t.Fatalf("expected arm64, got %s", info.Arch)
		}
	}
}

func TestDownloadPath(t *testing.T) {
	tests := []struct {
		version  string
		info     Info
		expected string
	}{
		{
			version:  "3.14.0",
			info:     Info{OS: "linux", Arch: "amd64"},
			expected: "helm-v3.14.0-linux-amd64.tar.gz",
		},
		{
			version:  "3.20.0",
			info:     Info{OS: "darwin", Arch: "arm64"},
			expected: "helm-v3.20.0-darwin-arm64.tar.gz",
		},
		{
			version:  "3.13.3",
			info:     Info{OS: "linux", Arch: "arm64"},
			expected: "helm-v3.13.3-linux-arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.info.OS+"_"+tt.info.Arch, func(t *testing.T) {
			path := DownloadPath(tt.version, tt.info)
			if path != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, path)
			}
		})
	}
}

func TestChecksumPath(t *testing.T) {
	info := Info{OS: "linux", Arch: "amd64"}
	got := ChecksumPath("3.14.0", info)
	expected := "helm-v3.14.0-linux-amd64.tar.gz.sha256sum"
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

func TestArchiveName(t *testing.T) {
	info := Info{OS: "darwin", Arch: "arm64"}
	got := ArchiveName("3.20.0", info)
	expected := "helm-v3.20.0-darwin-arm64.tar.gz"
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}
