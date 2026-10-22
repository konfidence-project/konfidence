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
			got, err := NewArtifactType(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewArtifactType(%q) error = %v, wantErr %t", tt.value, err, tt.wantErr)
			}
			if !tt.wantErr && got != ArtifactType(tt.value) {
				t.Errorf("NewArtifactType(%q) = %q, want %q", tt.value, got, tt.value)
			}
		})
	}
}

func TestMustArtifactType(t *testing.T) {
	const value = "helm.konfidence.cloud"
	if got := MustArtifactType(value); got != value {
		t.Fatalf("MustArtifactType(%q) = %q, want %q", value, got, value)
	}

	t.Run("panics for invalid value", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("MustArtifactType did not panic for invalid value")
			}
		}()
		MustArtifactType("invalid")
	})
}

func TestArtifactTypeValid(t *testing.T) {
	tests := []struct {
		value ArtifactType
		want  bool
	}{
		{value: "helm.konfidence.cloud", want: true},
		{value: "1helm.example.com", want: true},
		{value: "helm.example", want: false},
		{value: "helm.example-.com", want: false},
		{value: "", want: false},
	}

	for _, tt := range tests {
		if got := tt.value.Valid(); got != tt.want {
			t.Errorf("ArtifactType(%q).Valid() = %t, want %t", tt.value, got, tt.want)
		}
	}
}
