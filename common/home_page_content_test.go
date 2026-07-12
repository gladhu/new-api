package common

import "testing"

func TestNormalizeHomePageContentSource(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		serverAddress  string
		want           string
	}{
		{
			name:          "absolute server url becomes relative path",
			source:        "https://www.faceapi.ai/home-custom.html",
			serverAddress: "https://www.faceapi.ai",
			want:          "/home-custom.html",
		},
		{
			name:          "relative path unchanged",
			source:        "/home-custom.html",
			serverAddress: "https://www.faceapi.ai",
			want:          "/home-custom.html",
		},
		{
			name:          "external iframe url unchanged",
			source:        "https://example.com/page",
			serverAddress: "https://www.faceapi.ai",
			want:          "https://example.com/page",
		},
		{
			name:          "markdown content unchanged",
			source:        "# Welcome",
			serverAddress: "https://www.faceapi.ai",
			want:          "# Welcome",
		},
		{
			name:          "server url with query preserved",
			source:        "https://www.faceapi.ai/home-custom.html?theme=dark",
			serverAddress: "https://www.faceapi.ai/",
			want:          "/home-custom.html?theme=dark",
		},
		{
			name:          "empty source unchanged",
			source:        "   ",
			serverAddress: "https://www.faceapi.ai",
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeHomePageContentSource(tt.source, tt.serverAddress)
			if got != tt.want {
				t.Fatalf("NormalizeHomePageContentSource() = %q, want %q", got, tt.want)
			}
		})
	}
}
