package util

import "testing"

func TestBuildAllowedExtSet(t *testing.T) {
	set := buildAllowedExtSet(".jpg, .jpeg,.png")
	if _, ok := set[".jpg"]; !ok {
		t.Fatalf("expected .jpg in set")
	}
	if _, ok := set[".jpeg"]; !ok {
		t.Fatalf("expected .jpeg in set")
	}
	if _, ok := set[".png"]; !ok {
		t.Fatalf("expected .png in set")
	}
	if _, ok := set[".jp"]; ok {
		t.Fatalf("did not expect partial extension match")
	}
}

func TestNormalizeMIMEType(t *testing.T) {
	got := normalizeMIMEType("IMAGE/JPEG; charset=binary")
	if got != "image/jpeg" {
		t.Fatalf("normalizeMIMEType result mismatch: %s", got)
	}
}

func TestIsMIMETypeAllowed(t *testing.T) {
	tests := []struct {
		name      string
		ext       string
		mime      string
		allowType string
		want      bool
	}{
		{name: "image valid", ext: ".jpg", mime: "image/jpeg", allowType: AllowImageType, want: true},
		{name: "image invalid", ext: ".jpg", mime: "text/plain", allowType: AllowImageType, want: false},
		{name: "video valid", ext: ".mp4", mime: "video/mp4", allowType: AllowVideoType, want: true},
		{name: "video invalid", ext: ".mp4", mime: "image/png", allowType: AllowVideoType, want: false},
		{name: "mkv octet allowed", ext: ".mkv", mime: "application/octet-stream", allowType: AllowVideoType, want: true},
		{name: "mp4 octet denied", ext: ".mp4", mime: "application/octet-stream", allowType: AllowVideoType, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMIMETypeAllowed(tt.ext, tt.mime, tt.allowType)
			if got != tt.want {
				t.Fatalf("isMIMETypeAllowed(%s, %s, %s) = %v, want %v", tt.ext, tt.mime, tt.allowType, got, tt.want)
			}
		})
	}
}
