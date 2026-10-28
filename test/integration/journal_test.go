package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	zapjournal "github.com/black-desk/zap-journal"
	"github.com/black-desk/zap-journal/conn"
	"github.com/black-desk/zap-journal/encoder"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestJournalIntegration(t *testing.T) {
	work, err := os.MkdirTemp("", "zap-journal-integration-")
	if err != nil {
		t.Fatal(err)
	}
	// Also cover setup failures before the helper takes ownership.
	t.Cleanup(func() { _ = os.RemoveAll(work) })
	for _, name := range []string{"socket", "journal"} {
		if err := os.Mkdir(filepath.Join(work, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	check := exec.Command("sh", "helper.sh", "--check", work)
	if output, err := check.CombinedOutput(); err != nil {
		t.Skipf("journald prerequisite check failed: %v\n%s", err, output)
	}
	log, err := os.Create(filepath.Join(work, "journald.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	// Only journald runs in namespaces. The socket and journal directories
	// are bind-mounted from work so the client and journalctl stay outside.
	daemon := exec.Command("sh", "helper.sh", work)
	daemon.Stdout, daemon.Stderr = log, log
	stdin, err := daemon.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := daemon.Start(); err != nil {
		_ = stdin.Close()
		t.Fatalf("cannot start isolated journald: %v", err)
	}
	done := make(chan struct{})
	var daemonErr error
	go func() {
		daemonErr = daemon.Wait()
		close(done)
	}()
	defer func() {
		// Closing stdin stops the helper. Process exit also closes this pipe
		// if go test times out before this defer can run.
		_ = stdin.Close()
		<-done
		if _, err := os.Stat(work); err == nil {
			t.Error("helper did not remove its temporary directory")
		} else if !os.IsNotExist(err) {
			t.Errorf("check helper cleanup: %v", err)
		}
		if t.Failed() {
			data, _ := io.ReadAll(io.NewSectionReader(log, 0, 1<<63-1))
			t.Logf("journald output:\n%s", data)
		}
	}()
	address := filepath.Join(work, "socket", "socket")
	for {
		select {
		case <-done:
			data, _ := io.ReadAll(io.NewSectionReader(log, 0, 1<<63-1))
			t.Fatalf("helper exited before the journal socket was ready: %v\n%s", daemonErr, data)
		default:
		}
		if info, err := os.Stat(address); err == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Run("missing-socket", func(t *testing.T) {
		cfg := zapjournal.Config()
		cfg.OutputPaths = []string{"journal://" + address + ".missing"}
		cfg.ErrorOutputPaths = []string{"stderr"}
		logger, err := cfg.Build()
		if logger != nil {
			_ = logger.Sync()
			t.Fatal("Build returned a logger for a missing journal socket")
		}
		if err == nil {
			t.Fatal("Build must reject a missing journal socket")
		}
	})
	for _, tc := range []struct {
		name string
		size int
	}{
		{"datagram", 32},
		{"file-descriptor", 128 * 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			journal, err := conn.New(conn.WithAddress(address), conn.WithLimit(1024))
			if err != nil {
				t.Fatal(err)
			}
			defer journal.Close()
			enc, err := encoder.New(encoder.WithCfg(zapjournal.Config().EncoderConfig))
			if err != nil {
				t.Fatal(err)
			}
			var writeErrors bytes.Buffer
			logger := zap.New(zapcore.NewCore(enc, journal, zap.InfoLevel),
				zap.ErrorOutput(zapcore.AddSync(&writeErrors))).Named("zap-journal-integration")
			message := "zap-journal-integration-" + tc.name
			payload := strings.Repeat("x", tc.size) + "\nsecond line"
			logger.Info(message, zap.String("payload", payload))
			if err := logger.Sync(); err != nil {
				t.Fatal(err)
			}
			if writeErrors.Len() != 0 {
				t.Fatalf("zap write failed: %s", &writeErrors)
			}
			var output []byte
			for {
				cmd := exec.Command("sh", "helper.sh", "--read", work, "MESSAGE="+message)
				output, err = cmd.CombinedOutput()
				if err == nil && len(bytes.TrimSpace(output)) != 0 {
					var entry map[string]any
					if err := json.Unmarshal(bytes.TrimSpace(output), &entry); err != nil {
						t.Fatalf("decode journal entry: %v\n%.512s", err, output)
					}
					if entry["MESSAGE"] != message || entry["PAYLOAD"] != payload || entry["PRIORITY"] != "6" || entry["SYSLOG_IDENTIFIER"] != "zap-journal-integration" {
						t.Fatal("journal entry does not match the sent record")
					}
					return
				}
				if err != nil {
					t.Fatalf("journalctl failed: %v\n%.512s", err, output)
				}
				select {
				case <-done:
					t.Fatalf("journald exited: %v", daemonErr)
				case <-time.After(20 * time.Millisecond):
				}
			}
		})
	}
}
