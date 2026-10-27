package encoder_test

import (
	"fmt"
	"log/syslog"
	"testing"

	"github.com/black-desk/zap-journal/encoder"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Systemd journal encoder for zap", func() {
	describeEncoder := func(constructor func() zapcore.Encoder) {
		var enc zapcore.Encoder
		BeforeEach(func() {
			enc = constructor()
		})
		DescribeTable("encodes", func(field zapcore.Field, result []byte) {
			buf, err := enc.EncodeEntry(zapcore.Entry{}, []zapcore.Field{field})
			Expect(err).To(Succeed())
			defer buf.Free()
			Expect(buf.Bytes()).To(Equal(result))
		},
			Entry("an int", zap.Int32("SOME_INT", 1), []byte(fmt.Sprintf(""+
				"PRIORITY=%d\n"+
				"SYSLOG_FACILITY=%d\n"+
				"SYSLOG_IDENTIFIER\n"+
				"\000\000\000\000\000\000\000\000"+
				"\n"+
				"SOME_INT=%d\n",
				syslog.LOG_INFO, syslog.LOG_USER, 1,
			))),
			Entry("a string", zap.String("SOME_STRING", "a"), []byte(fmt.Sprintf(""+
				"PRIORITY=%d\n"+
				"SYSLOG_FACILITY=%d\n"+
				"SYSLOG_IDENTIFIER\n"+
				"\000\000\000\000\000\000\000\000"+
				"\n"+
				"SOME_STRING\n"+
				"\001\000\000\000\000\000\000\000"+
				"%s\n",
				syslog.LOG_INFO, syslog.LOG_USER, "a",
			))),
			Entry("a struct",
				zap.Reflect("SOME_STRUCT",
					struct {
						Field1 int
						Field2 string
					}{
						Field1: 1,
						Field2: "string content",
					}),
				[]byte(fmt.Sprintf(""+
					"PRIORITY=%d\n"+
					"SYSLOG_FACILITY=%d\n"+
					"SYSLOG_IDENTIFIER\n"+
					"\000\000\000\000\000\000\000\000"+
					"\n"+
					"SOME_STRUCT\n"+
					"\046\000\000\000\000\000\000\000"+
					"%s\n",
					syslog.LOG_INFO, syslog.LOG_USER,
					`{"Field1":1,"Field2":"string content"}`,
				)),
			),
			Entry("a map",
				zap.Reflect("SOME_MAP",
					map[string]string{
						"key1": "value1",
						"key2": "value2",
						"key3": "value3",
					}),
				[]byte(fmt.Sprintf(""+
					"PRIORITY=%d\n"+
					"SYSLOG_FACILITY=%d\n"+
					"SYSLOG_IDENTIFIER\n"+
					"\000\000\000\000\000\000\000\000"+
					"\n"+
					"SOME_MAP\n\061\000\000\000\000\000\000\000%s\n",
					syslog.LOG_INFO, syslog.LOG_USER,
					`{"key1":"value1","key2":"value2","key3":"value3"}`,
				)),
			),
			Entry("an array",
				zap.Reflect("SOME_ARRAY",
					[]string{"string1", "string2", "string3"}),
				[]byte(fmt.Sprintf(""+
					"PRIORITY=%d\n"+
					"SYSLOG_FACILITY=%d\n"+
					"SYSLOG_IDENTIFIER\n"+
					"\000\000\000\000\000\000\000\000"+
					"\n"+
					"SOME_ARRAY\n"+
					"\037\000\000\000\000\000\000\000"+
					"%s\n",
					syslog.LOG_INFO, syslog.LOG_USER,
					`["string1","string2","string3"]`,
				)),
			),
		)
	}
	Context("created with New", func() {
		describeEncoder(func() zapcore.Encoder {
			enc, err := encoder.New(
				encoder.WithCfg(zap.NewProductionEncoderConfig()),
			)
			Expect(err).To(Succeed())
			return enc
		})
	})
	Context("created with NewJournalEncoder", func() {
		describeEncoder(func() zapcore.Encoder {
			enc, err := encoder.NewJournalEncoder(zap.NewProductionEncoderConfig())
			Expect(err).To(Succeed())
			return enc
		})
	})
})

func TestJournal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Systemd-journal plugin for zap Suite")
}
