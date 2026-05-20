/*
Copyright 2026.

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

package controller

import (
	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("parsePromotionParameters", func() {
	Context("with valid references", func() {
		It("returns parsed source and target refs", func() {
			config := &global.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
				Spec: global.VectorPromotionConfigSpec{
					Source: "ghcr.io/org/components//github.com/org/app:1.0.0",
					Target: "ghcr.io/org/components//github.com/org/app:production",
				},
			}

			src, dst, err := parsePromotionParameters(config)

			Expect(err).ToNot(HaveOccurred())
			Expect(src).ToNot(BeNil())
			Expect(src.Component).To(Equal("github.com/org/app"))
			Expect(src.Version).To(Equal("1.0.0"))
			Expect(dst).ToNot(BeNil())
			Expect(dst.Component).To(Equal("github.com/org/app"))
			Expect(dst.Version).To(Equal("production"))
		})
	})

	Context("with invalid source", func() {
		It("returns error for malformed source reference", func() {
			config := &global.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
				Spec: global.VectorPromotionConfigSpec{
					Source: "not-a-valid-reference",
					Target: "ghcr.io/org/components//github.com/org/app:production",
				},
			}

			_, _, err := parsePromotionParameters(config)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse source reference"))
		})

		It("returns error for source without version", func() {
			config := &global.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
				Spec: global.VectorPromotionConfigSpec{
					Source: "ghcr.io/org/components//github.com/org/app",
					Target: "ghcr.io/org/components//github.com/org/app:production",
				},
			}

			_, _, err := parsePromotionParameters(config)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse source reference"))
		})
	})

	Context("with mismatched component names", func() {
		It("returns error when source and target components differ", func() {
			config := &global.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
				Spec: global.VectorPromotionConfigSpec{
					Source: "ghcr.io/org/components//github.com/org/app-a:1.0.0",
					Target: "ghcr.io/org/components//github.com/org/app-b:production",
				},
			}

			_, _, err := parsePromotionParameters(config)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source and target component names do not match"))
		})
	})

	Context("with invalid target", func() {
		It("returns error for target with semver version", func() {
			config := &global.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
				Spec: global.VectorPromotionConfigSpec{
					Source: "ghcr.io/org/components//github.com/org/app:1.0.0",
					Target: "ghcr.io/org/components//github.com/org/app:2.0.0",
				},
			}

			_, _, err := parsePromotionParameters(config)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse target reference"))
		})

		It("returns error for malformed target reference", func() {
			config := &global.VectorPromotionConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
				Spec: global.VectorPromotionConfigSpec{
					Source: "ghcr.io/org/components//github.com/org/app:1.0.0",
					Target: "not-a-valid-reference",
				},
			}

			_, _, err := parsePromotionParameters(config)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse target reference"))
		})
	})
})
