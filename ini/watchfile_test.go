package ini_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/thrawn01/args"
	"github.com/thrawn01/args/ini"
)

func saveFile(fileName string, content []byte) error {
	err := ioutil.WriteFile(fileName, content, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "Failed to write '%s'", fileName)
	}
	return nil
}

var _ = Describe("args.WatchFile()", func() {
	var log *TestLogger
	iniVersion1 := []byte(`
	    help=false
		value=my-value
		version=1
	`)
	iniVersion2 := []byte(`
	    help=false
		value=new-value
		version=2
	`)

	iniDeleted := []byte(`
	    help=false
		version=1
	`)

	iniAdded := []byte(`
	    help=false
		value=my-value
		version=1
		new=thing
	`)

	BeforeEach(func() {
		log = NewTestLogger()
	})

	Context("while watching a file", func() {
		var parser *args.Parser

		BeforeEach(func() {
			parser = args.NewParser()
			parser.Log(log)

			parser.AddConfig("value")
			parser.AddConfig("new")
			parser.AddConfig("version").IsInt()

			opt, err := parser.Parse(nil)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opt.String("value")).To(Equal(""))
			Expect(opt.Int("version")).To(Equal(0))
		})

		It("Should reload a config file when watched file is modified", func() {
			iniFile, err := ioutil.TempFile("/tmp", "args-test")
			if err != nil {
				Fail(err.Error())
			}
			defer os.Remove(iniFile.Name())

			// Write version 1 of the ini file
			if err := saveFile(iniFile.Name(), iniVersion1); err != nil {
				Fail(err.Error())
			}
			iniFile.Close()

			// Load the INI file
			content, err := args.LoadFile(iniFile.Name())
			if err != nil {
				Fail(err.Error())
			}
			// Parse the ini file
			backend, err := ini.NewBackend(content, iniFile.Name())

			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("value")).To(Equal("my-value"))
			Expect(opts.Int("version")).To(Equal(1))

			// Should see 2 change events occur
			wg := &sync.WaitGroup{}
			wg.Add(2)

			cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
				parser.Apply(opts.FromChangeEvent(event))
				wg.Done()
			})
			if err != nil {
				Fail(err.Error())
			}

			if err := saveFile(iniFile.Name(), iniVersion2); err != nil {
				Fail(err.Error())
			}
			// Wait until all change events processed
			wg.Wait()
			// Stop the watch
			cancelWatch()
			// Get the updated options
			opts = parser.GetOpts()

			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("value")).To(Equal("new-value"))
			Expect(opts.Int("version")).To(Equal(2))
		})

		It("Should report deleted values when file is modified", func() {
			iniFile, err := ioutil.TempFile("/tmp", "args-test")
			if err != nil {
				Fail(err.Error())
			}
			defer os.Remove(iniFile.Name())

			// Write version 1 of the ini file
			if err := saveFile(iniFile.Name(), iniVersion1); err != nil {
				Fail(err.Error())
			}
			iniFile.Close()

			// Load the INI file
			content, err := args.LoadFile(iniFile.Name())
			if err != nil {
				Fail(err.Error())
			}
			// Parse the ini file
			backend, err := ini.NewBackend(content, iniFile.Name())

			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("value")).To(Equal("my-value"))
			Expect(opts.Int("version")).To(Equal(1))

			// Should see 1 change event occur
			wg := &sync.WaitGroup{}
			wg.Add(1)

			cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
				Expect(event.Deleted).To(Equal(true))
				parser.Apply(opts.FromChangeEvent(event))
				wg.Done()
			})
			if err != nil {
				Fail(err.Error())
			}

			if err := saveFile(iniFile.Name(), iniDeleted); err != nil {
				Fail(err.Error())
			}
			// Wait until all change events processed
			wg.Wait()
			// Stop the watch
			cancelWatch()
			// Get the updated options
			opts = parser.GetOpts()

			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("value")).To(Equal(""))
			Expect(opts.Int("version")).To(Equal(1))
		})

		It("Should report added values when file is modified", func() {
			iniFile, err := ioutil.TempFile("/tmp", "args-test")
			if err != nil {
				Fail(err.Error())
			}
			defer os.Remove(iniFile.Name())

			// Write version 1 of the ini file
			if err := saveFile(iniFile.Name(), iniVersion1); err != nil {
				Fail(err.Error())
			}
			iniFile.Close()

			// Load the INI file
			content, err := args.LoadFile(iniFile.Name())
			if err != nil {
				Fail(err.Error())
			}
			// Parse the ini file
			backend, err := ini.NewBackend(content, iniFile.Name())

			opts, err := parser.FromBackend(backend)
			Expect(err).To(BeNil())
			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("value")).To(Equal("my-value"))
			Expect(opts.String("new")).To(Equal(""))
			Expect(opts.Int("version")).To(Equal(1))

			// Should see 1 change event occur
			wg := &sync.WaitGroup{}
			wg.Add(1)

			cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
				parser.Apply(opts.FromChangeEvent(event))
				wg.Done()
			})
			if err != nil {
				Fail(err.Error())
			}

			if err := saveFile(iniFile.Name(), iniAdded); err != nil {
				Fail(err.Error())
			}
			// Wait until all change events processed
			wg.Wait()
			// Stop the watch
			cancelWatch()
			// Get the updated options
			opts = parser.GetOpts()

			Expect(log.GetEntry()).To(Equal(""))
			Expect(opts.String("value")).To(Equal("my-value"))
			Expect(opts.String("new")).To(Equal("thing"))
			Expect(opts.Int("version")).To(Equal(1))
		})
		It("Should signal a modification if the file is deleted and re-created", func() {
			iniFile, err := ioutil.TempFile("/tmp", "args-test")
			if err != nil {
				Fail(err.Error())
			}

			// Write version 1 of the ini file
			if err := saveFile(iniFile.Name(), iniVersion1); err != nil {
				Fail(err.Error())
			}
			iniFile.Close()

			// Load the INI file
			content, err := args.LoadFile(iniFile.Name())
			if err != nil {
				Fail(err.Error())
			}
			// Parse the ini file
			backend, err := ini.NewBackend(content, iniFile.Name())

			_, err = parser.FromBackend(backend)
			Expect(err).To(BeNil())

			var watchErr error
			wg := &sync.WaitGroup{}
			wg.Add(2)

			cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
				if err != nil {
					fmt.Printf("Watch Error %s\n", err.Error())
				}
				wg.Done()
			})
			if err != nil {
				Fail(err.Error())
			}

			// Quickly Remove the file and replace it
			os.Remove(iniFile.Name())
			if err := saveFile(iniFile.Name(), iniVersion2); err != nil {
				Fail(err.Error())
			}
			//iniFile, err = os.OpenFile(iniFile.Name(), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
			iniFile.Close()
			defer os.Remove(iniFile.Name())

			// Wait until the all the change events have processed
			wg.Wait()
			Expect(watchErr).To(BeNil())

			// Stop the watch
			cancelWatch()
		})

		// Apparently VIM does this to files on OSX
		It("Should signal a modification if the file is renamed and renamed back", func() {
			iniFile, err := ioutil.TempFile("/tmp", "args-test")
			if err != nil {
				Fail(err.Error())
			}

			// Write version 1 of the ini file
			if err := saveFile(iniFile.Name(), iniVersion1); err != nil {
				Fail(err.Error())
			}
			iniFile.Close()

			// Load the INI file
			content, err := args.LoadFile(iniFile.Name())
			if err != nil {
				Fail(err.Error())
			}
			// Parse the ini file
			backend, err := ini.NewBackend(content, iniFile.Name())

			_, err = parser.FromBackend(backend)
			Expect(err).To(BeNil())

			var watchErr error
			wg := &sync.WaitGroup{}
			wg.Add(2)

			cancelWatch := parser.Watch(backend, func(event args.ChangeEvent, err error) {
				if err != nil {
					fmt.Printf("Watch Error %s\n", err.Error())
				}
				wg.Done()
			})
			if err != nil {
				Fail(err.Error())
			}

			// Quickly Remove the file and replace it
			os.Rename(iniFile.Name(), iniFile.Name()+"-new")
			os.Rename(iniFile.Name()+"-new", iniFile.Name())
			if err := saveFile(iniFile.Name(), iniVersion2); err != nil {
				Fail(err.Error())
			}
			defer os.Remove(iniFile.Name())

			// Wait until the new file was loaded
			wg.Wait()
			Expect(watchErr).To(BeNil())
			// Stop the watch
			cancelWatch()
		})
	})
})
