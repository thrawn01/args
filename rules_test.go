package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
)

var _ = Describe("Rule", func() {
	Describe("Rule.SetFlags()", func() {
		It("Should set the proper flags", func() {
			rule := &args.Rule{}
			rule.SetFlag(args.WasSeenInArgv)
			Expect(rule.HasFlag(args.WasSeenInArgv)).To(Equal(true))
		})
	})

	Describe("Rule.ClearFlags()", func() {
		It("Should clear flags", func() {
			rule := &args.Rule{}
			rule.SetFlag(args.WasSeenInArgv)
			rule.ClearFlag(args.WasSeenInArgv)
			Expect(rule.HasFlag(args.WasSeenInArgv)).To(Equal(false))
			// Regression, ClearFlags was not clearing the flag, just rotating it
			rule.ClearFlag(args.WasSeenInArgv)
			Expect(rule.HasFlag(args.WasSeenInArgv)).To(Equal(false))
		})
	})
})
