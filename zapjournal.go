package zapjournal

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/black-desk/zap-journal/conn"
	_ "github.com/black-desk/zap-journal/encoder"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Config() zap.Config {
	return zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "journal",
		EncoderConfig: zapcore.EncoderConfig{
			StacktraceKey:  "STACKTRACE",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		},
		OutputPaths:      []string{"journal:///run/systemd/journal/socket"},
		ErrorOutputPaths: []string{"journal:///run/systemd/journal/socket"},
	}
}

func DebugConfig() zap.Config {
	return zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
		Development: true,
		Encoding:    "journal",
		EncoderConfig: zapcore.EncoderConfig{
			StacktraceKey:  "STACKTRACE",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		},
		OutputPaths:      []string{"journal:///run/systemd/journal/socket"},
		ErrorOutputPaths: []string{"journal:///run/systemd/journal/socket"},
	}
}

func New(opts ...zap.Option) (ret *zap.Logger, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(
			"failed to create new journald logger: %w",
			err,
		)
	}()

	var log *zap.Logger
	log, err = Config().Build()
	if err != nil {
		return
	}

	log = log.WithOptions(opts...)

	var exe string
	exe, err = os.Executable()
	if err != nil {
		err = fmt.Errorf("failed to get exe: %w", err)
		return
	}
	log = log.Named(filepath.Base(exe))

	ret = log

	return
}

func NewDebug(opts ...zap.Option) (ret *zap.Logger, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(
			"failed to create new journald logger: %w",
			err,
		)
	}()

	var log *zap.Logger
	log, err = DebugConfig().Build()
	if err != nil {
		return
	}

	log = log.WithOptions(opts...)

	var exe string
	exe, err = os.Executable()
	if err != nil {
		err = fmt.Errorf("failed to get exe: %w", err)
		return
	}
	log = log.Named(filepath.Base(exe))

	ret = log

	return
}
