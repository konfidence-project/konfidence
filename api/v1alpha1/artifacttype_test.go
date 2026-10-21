package v1alpha1

import "testing"

func TestNewArtifactType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "type with multi-label domain", value: "helm.konfidence.cloud"},
		{name: "single character labels", value: "a.b.c"},
		{name: "digits and internal hyphens", value: "h-1.v2.example-domain"},
		{name: "type label may start with digit", value: "1helm.example.com"},
		{name: "missing domain label", value: "helm.example", wantErr: true},
		{name: "leading hyphen", value: "-helm.example.com", wantErr: true},
		{name: "trailing hyphen", value: "helm.example-.com", wantErr: true},
		{name: "uppercase", value: "Helm.example.com", wantErr: true},
		{name: "whitespace", value: " helm.example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewArtifactType(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewArtifactType(%q) error = %v, wantErr %t", tt.value, err, tt.wantErr)
			}
		})
	}
}
