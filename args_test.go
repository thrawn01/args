package args_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thrawn01/args"
	"testing"
)

func TestArgs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Args Parser")
}

var _ = Describe("Opt()", func() {
	/*BeforeEach(func() {
	})*/

	Describe("Should provide valid config", func() {
		Describe("When", func() {
			Context("blah blah", func() {
				It("blah blah", func() {
					parser := args.Parser()
					parser.Opt("argument", args.Alias("-a"))
				})
			})
		})
	})
})
