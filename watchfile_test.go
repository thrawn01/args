package args_test

import (
	"time"

	"io/ioutil"

	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/thrawn01/args"
)

func loadFile(fileName string) ([]byte, error) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read '%s'", fileName)
	}
	return content, nil
}

func saveFile(fileName string, content []byte) error {
	err := ioutil.WriteFile(fileName, content, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "Failed to write '%s'", fileName)
	}
	return nil
}

var _ = Describe("ArgParser.WatchFile()", func() {
	var iniFile *os.File
	var log *TestLogger
	iniVersion1 := []byte(`
		value=my-value
		version=1
	`)
	iniVersion2 := []byte(`
		value=new-value
		version=2
	`)

	BeforeEach(func() {
		log = NewTestLogger()
	})

	It("Should call Watch() to watch for new values", func() {
		parser := args.NewParser()
		parser.SetLog(log)

		parser.AddConfig("value")
		parser.AddConfig("version").IsInt()

		opt, err := parser.ParseArgs(nil)
		Expect(err).To(BeNil())
		Expect(log.GetEntry()).To(Equal(""))
		Expect(opt.String("value")).To(Equal(""))
		Expect(opt.Int("version")).To(Equal(0))

		iniFile, err = ioutil.TempFile("/tmp", "args-test")
		if err != nil {
			Fail(err.Error())
		}
		defer os.Remove(iniFile.Name())

		// Write version 1 of the ini file
		if err := saveFile(iniFile.Name(), iniVersion1); err != nil {
			Fail(err.Error())
		}

		// Load the INI file
		content, err := loadFile(iniFile.Name())
		if err != nil {
			Fail(err.Error())
		}
		// Parse the ini file
		opt, err = parser.FromIni(content)
		Expect(err).To(BeNil())
		Expect(log.GetEntry()).To(Equal(""))
		Expect(opt.String("value")).To(Equal("my-value"))
		Expect(opt.Int("version")).To(Equal(1))

		done := make(chan struct{})
		cancelWatch, err := args.WatchFile(iniFile.Name(), time.Second, func(err error) {
			content, err := loadFile(iniFile.Name())
			if err != nil {
				Fail(err.Error())
			}
			parser.FromIni(content)
			// Tell the test to continue, Change event was handled
			close(done)
		})
		if err != nil {
			Fail(err.Error())
		}

		if err := saveFile(iniFile.Name(), iniVersion2); err != nil {
			Fail(err.Error())
		}
		// Wait until the new file was loaded
		<-done
		// Stop the watch
		cancelWatch()
		// Get the updated options
		opts := parser.GetOpts()

		Expect(log.GetEntry()).To(Equal(""))
		Expect(opts.String("value")).To(Equal("new-value"))
		Expect(opts.Int("version")).To(Equal(2))
	})
})
