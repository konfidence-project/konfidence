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
	"net/url"
	"strings"
)

// ExtractHostname extracts the hostname without port number from a url
func ExtractHostname(rawURL string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "//" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	host := u.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host, nil
}
