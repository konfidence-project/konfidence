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

package sanitize

import (
	"regexp"
	"strings"
)

// DNSLabelName makes a string RFC 1123 DNS Label name compatible (valid K8s resource name)
func DNSLabelName(name string) string {
	return sanitizeName(name, `[^a-z0-9-]`, 63)
}

// DNSSubdomainName makes a string RFC 1123 DNS subdomain name compatible (valid K8s resource name)
func DNSSubdomainName(name string) string {
	return sanitizeName(name, `[^a-z0-9-.]`, 253)
}

// ResourceName makes a string RFC 1123 DNS subdomain name compatible but only allows dashes and not dots
func ResourceName(name string) string {
	return sanitizeName(name, `[^a-z0-9-]`, 253)
}

func sanitizeName(name string, pattern string, maxLen int) string {
	// lowercase the name
	name = strings.ToLower(name)

	// replace any character not allowed with a dash
	reg := regexp.MustCompile(pattern)
	name = reg.ReplaceAllString(name, "-")

	// trim leading/trailing non-alphanumeric characters
	name = strings.TrimLeftFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	name = strings.TrimRightFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})

	// truncate to maxLen characters max
	if len(name) > maxLen {
		name = name[:maxLen]
	}

	return name
}
