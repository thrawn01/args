package args

import (
	"fmt"

	"github.com/go-ini/ini"
	"github.com/pkg/errors"
)

// Parse the INI file and the Apply() the values to the parser
func (self *ArgParser) FromIni(input []byte) (*Options, error) {
	options, err := self.ParseIni(input)
	if err != nil {
		return options, err
	}
	// Apply the ini file values to the commandline and environment variables
	return self.Apply(options)
}

func (self *ArgParser) FromIniFile(fileName string) (*Options, error) {
	content, err := LoadFile(fileName)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("'%s'", fileName))
	}
	return self.FromIni(content)
}

// Parse the INI file and return the raw parsed options
func (self *ArgParser) ParseIni(input []byte) (*Options, error) {
	// Parse the file return a map of the contents
	cfg, err := ini.Load(input)
	if err != nil {
		return nil, err
	}
	values := self.NewOptions()
	for _, section := range cfg.Sections() {
		group := cfg.Section(section.Name())
		for _, key := range group.KeyStrings() {
			// Always use our default option group name for the DEFAULT section
			name := section.Name()
			if name == "DEFAULT" {
				name = DefaultOptionGroup
			}
			values.Group(name).Set(key, group.Key(key).String())
		}

	}
	return values, nil
}
