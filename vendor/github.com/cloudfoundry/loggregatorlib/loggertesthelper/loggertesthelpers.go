package loggertesthelper

import (
	"os"
	"strings"
	"sync"

	"github.com/cloudfoundry/gosteno"
)

var TestLoggerSink = new(TestStenoSink)
var logger = getLogger(false)

func StdOutLogger() *gosteno.Logger {
	return getLogger(true)
}

func Logger() *gosteno.Logger {
	return logger
}

func getLogger(debug bool) *gosteno.Logger {

	level := gosteno.LOG_DEBUG

	loggingConfig := &gosteno.Config{
		Sinks:     []gosteno.Sink{TestLoggerSink},
		Level:     level,
		Codec:     gosteno.NewJsonCodec(),
		EnableLOC: true,
	}

	if debug {
		loggingConfig.Sinks[0] = gosteno.NewIOSink(os.Stdout)
	}

	gosteno.Init(loggingConfig)

	return gosteno.NewLogger("TestLogger")
}

type TestStenoSink struct {
	sync.Mutex
	records []*gosteno.Record
	codec   gosteno.Codec
}

func (t *TestStenoSink) AddRecord(record *gosteno.Record) {
	t.Lock()
	defer t.Unlock()
	t.records = append(t.records, record)
}
func (t *TestStenoSink) Flush() {}

func (t *TestStenoSink) SetCodec(codec gosteno.Codec) {
	t.codec = codec
}

func (t *TestStenoSink) GetCodec() gosteno.Codec {
	return t.codec
}

func (t *TestStenoSink) LogContents() string {
	t.Lock()
	defer t.Unlock()

	data := make([]string, len(t.records))
	for i, record := range t.records {
		data[i] = record.Message
	}
	return strings.Join(data, "\n")
}

func (t *TestStenoSink) Clear() {
	t.Lock()
	defer t.Unlock()

	t.records = make([]*gosteno.Record, 0, 20)
}
