/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package url

import (
	"testing"
)

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full url",
			input:    "https://user@test.registry.com:5100/v2/ocm/repository",
			expected: "test.registry.com",
		},
		{
			name:     "simple host with www prefix and no port",
			input:    "www.registry.com",
			expected: "registry.com",
		},
		{
			name:     "invalid url",
			input:    "?www.registry.com",
			expected: "",
		},
		{
			name:     "url with some dashes and query param",
			input:    "http://registry-ocm.test/test?id=123",
			expected: "registry-ocm.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := ExtractHostname(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractHostname(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractHostnameWithOptionalPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full url",
			input:    "https://user@test.registry.com:5100/v2/ocm/repository",
			expected: "test.registry.com:5100",
		},
		{
			name:     "simple host with www prefix and no port",
			input:    "www.registry.com",
			expected: "registry.com",
		},
		{
			name:     "invalid url",
			input:    "?www.registry.com",
			expected: "",
		},
		{
			name:     "url with some dashes and query param",
			input:    "http://registry-ocm.test:8080/test?id=123",
			expected: "registry-ocm.test:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := ExtractHostnameWithOptionalPort(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractHostnameWithOptionalPort(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
