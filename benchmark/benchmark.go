package benchmark

import (
	"fmt"
	"net/http"
	"sync"
)

type OptValues map[string]string
type Options struct {
	values OptValues
}

func (o *Options) Get(key string) string {
	return o.values[key]
}

type ArgParser interface {
	SetOpts(values OptValues)
	GetOpts() *Options
}

type Api struct {
	Parser ArgParser
}

func (s *Api) NewServer() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("/", s.Index)
	return router
}

func (s *Api) Index(resp http.ResponseWriter, req *http.Request) {
	opts := s.Parser.GetOpts()
	fmt.Fprintf(resp, "TestValue: %s", opts.Get("test-value"))
}

// No Thread Safety
type ParserBench struct {
	opts *Options
}

// =================================================================
func NewParserBench(values OptValues) *ParserBench {
	parser := &ParserBench{&Options{}}
	parser.SetOpts(values)
	return parser
}

func (pb *ParserBench) SetOpts(values OptValues) {
	pb.opts = &Options{values}
}

func (pb *ParserBench) GetOpts() *Options {
	return pb.opts
}

type ParserBenchMutex struct {
	opts  *Options
	mutex *sync.Mutex
}

// =================================================================
func NewParserBenchMutex(values OptValues) *ParserBenchMutex {
	parser := &ParserBenchMutex{&Options{}, &sync.Mutex{}}
	parser.SetOpts(values)
	return parser
}

func (pbm *ParserBenchMutex) SetOpts(values OptValues) {
	pbm.mutex.Lock()
	pbm.opts = &Options{values}
	pbm.mutex.Unlock()
}

func (pbm *ParserBenchMutex) GetOpts() *Options {
	pbm.mutex.Lock()
	defer func() {
		pbm.mutex.Unlock()
	}()
	return pbm.opts
}

// =================================================================
type ParserBenchRWMutex struct {
	opts  *Options
	mutex *sync.RWMutex
}

func NewParserBenchRWMutex(values OptValues) *ParserBenchRWMutex {
	parser := &ParserBenchRWMutex{&Options{}, &sync.RWMutex{}}
	parser.SetOpts(values)
	return parser
}

func (pbmr *ParserBenchRWMutex) SetOpts(values OptValues) {
	pbmr.mutex.Lock()
	pbmr.opts = &Options{values}
	pbmr.mutex.Unlock()
}

func (pbmr *ParserBenchRWMutex) GetOpts() *Options {
	pbmr.mutex.Lock()
	defer func() {
		pbmr.mutex.Unlock()
	}()
	return pbmr.opts
}

// =================================================================
type ParserBenchChannel struct {
	opts *Options
	get  chan *Options
	set  chan *Options
	done chan bool
}

func NewParserBenchChannel(values OptValues) *ParserBenchChannel {
	parser := &ParserBenchChannel{&Options{}, make(chan *Options), make(chan *Options), make(chan bool)}
	parser.Open()
	parser.SetOpts(values)
	return parser
}

func (pbc *ParserBenchChannel) Open() {
	go func() {
		defer func() {
			close(pbc.get)
			close(pbc.set)
			close(pbc.done)
		}()
		for {
			select {
			case pbc.get <- pbc.opts:
			case value := <-pbc.set:
				pbc.opts = value
			case <-pbc.done:
				return
			}
		}
	}()
}

func (pbc *ParserBenchChannel) Close() {
	pbc.done <- true
}

func (pbc *ParserBenchChannel) SetOpts(values OptValues) {
	pbc.set <- &Options{values}
}

func (pbc *ParserBenchChannel) GetOpts() *Options {
	return <-pbc.get
}
