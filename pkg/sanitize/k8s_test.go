package sanitize

import (
	"testing"
)

func TestSanitizeK8sDNSLabelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple valid name",
			input:    "my-resource",
			expected: "my-resource",
		},
		{
			name:     "uppercase letters",
			input:    "MyResource",
			expected: "myresource",
		},
		{
			name:     "special characters",
			input:    "my_resource@example.com",
			expected: "my-resource-example-com",
		},
		{
			name:     "leading and trailing invalid chars",
			input:    "_my-resource_",
			expected: "my-resource",
		},
		{
			name:     "name too long",
			input:    "this-is-a-very-long-name-that-exceeds-sixty-three-characters-limit",
			expected: "this-is-a-very-long-name-that-exceeds-sixty-three-characters-li",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple invalid characters",
			input:    "a@#b$%c",
			expected: "a--b--c",
		},
		{
			name:     "only invalid characters",
			input:    "@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DNSLabelName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeK8sDNSLabelName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeK8sDNSSubdomainName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple valid name with dashes",
			input:    "my-resource",
			expected: "my-resource",
		},
		{
			name:     "simple valid name with dots",
			input:    "test.resource.com",
			expected: "test.resource.com",
		},
		{
			name:     "simple valid name with dashes and dots",
			input:    "test.resource-123.com",
			expected: "test.resource-123.com",
		},
		{
			name:     "uppercase letters",
			input:    "MyResource",
			expected: "myresource",
		},
		{
			name:     "special characters",
			input:    "my_resource@example.com",
			expected: "my-resource-example.com",
		},
		{
			name:     "leading and trailing invalid chars",
			input:    "-my-resource_",
			expected: "my-resource",
		},
		{
			name: "name too long",
			input: "this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit",
			expected: "this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit.this-is-a-very-lo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple invalid characters",
			input:    "a@#b$%c",
			expected: "a--b--c",
		},
		{
			name:     "only invalid characters",
			input:    "@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DNSSubdomainName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeK8sDNSSubdomainName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResourceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple valid name with dashes",
			input:    "my-resource",
			expected: "my-resource",
		},
		{
			name:     "simple valid name with dots",
			input:    "test.resource.com",
			expected: "test-resource-com",
		},
		{
			name:     "simple valid name with dashes and dots",
			input:    "test.resource-123.com",
			expected: "test-resource-123-com",
		},
		{
			name:     "special characters",
			input:    "my_resource@example.com",
			expected: "my-resource-example-com",
		},
		{
			name:     "leading and trailing invalid chars",
			input:    "-my-resource_",
			expected: "my-resource",
		},
		{
			name: "name too long",
			input: "this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit." +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit",
			expected: "this-is-a-very-long-name-that-exceeds-253-characters-limit-" +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit-" +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit-" +
				"this-is-a-very-long-name-that-exceeds-253-characters-limit-this-is-a-very-lo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple invalid characters",
			input:    "a@#b$%c",
			expected: "a--b--c",
		},
		{
			name:     "only invalid characters",
			input:    "@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResourceName(tt.input)
			if result != tt.expected {
				t.Errorf("ResourceName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
