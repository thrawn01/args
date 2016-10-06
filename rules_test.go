package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

var _ = Describe("Rule", func() {
	Describe("Rule.UnEscape()", func() {
		It("Should unescape strings with black slash's", func() {
			rule := &args.Rule{}
			Expect(rule.UnEscape("\\-\\-help")).To(Equal("--help"))
			Expect(rule.UnEscape("--help")).To(Equal("--help"))
		})
	})

	Describe("Rule.SetFlags()", func() {
		It("Should set the proper flags", func() {
			rule := &args.Rule{}
			rule.SetFlags(args.Seen)
			Expect(rule.HasFlags(args.Seen)).To(Equal(true))
		})
	})

	Describe("Rule.ClearFlags()", func() {
		It("Should clear flags", func() {
			rule := &args.Rule{}
			rule.SetFlags(args.Seen)
			rule.ClearFlags(args.Seen)
			Expect(rule.HasFlags(args.Seen)).To(Equal(false))
			// Regression, ClearFlags was not clearing the flag, just rotating it
			rule.ClearFlags(args.Seen)
			Expect(rule.HasFlags(args.Seen)).To(Equal(false))
		})
	})
})
