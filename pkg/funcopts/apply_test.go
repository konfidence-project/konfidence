package funcopts_test

import (
	"errors"
	"github.com/konfidence-project/pkg/funcopts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type (
	Option func(instance *TestInstance)

	TestInstance struct {
		value bool
	}
)

func WithValue(value bool) Option {
	return func(instance *TestInstance) {
		instance.value = value
	}
}

var _ = Describe("Apply functional options", func() {
	Context("when applying options to an instance", func() {
		It("should apply the options correctly", func() {
			options := []Option{WithValue(true)}
			instance := funcopts.Apply[*TestInstance, Option](&TestInstance{}, options)
			Expect(instance.value).To(BeTrue())
		})
	})

	Context("when applying options with errors", func() {
		It("should apply options without errors", func() {
			options := []OptionWithError{WithError(nil)}
			instance, err := funcopts.ApplyWithErrors[*TestInstanceWithErr, OptionWithError](&TestInstanceWithErr{}, options)
			Expect(err).NotTo(HaveOccurred())
			Expect(instance).NotTo(BeNil())
		})

		It("should return an error if any option fails", func() {
			options := []OptionWithError{WithError(errors.New("test error"))}
			instance, err := funcopts.ApplyWithErrors[*TestInstanceWithErr, OptionWithError](&TestInstanceWithErr{}, options)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test error"))
			Expect(instance).To(BeNil())
		})
	})
})

type (
	OptionWithError func(instance *TestInstanceWithErr) error

	TestInstanceWithErr struct {
		value bool
	}
)

func WithError(err error) OptionWithError {
	return func(instance *TestInstanceWithErr) error {
		return err
	}
}
