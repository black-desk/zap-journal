// SPDX-FileCopyrightText: 2023-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: MIT

package conn

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"go.uber.org/zap"
)

var _ zap.Sink = &JournalConn{}

func (c *JournalConn) Write(p []byte) (n int, err error) {
	if c.limit != -1 && len(p) >= int(c.limit) {
		return c.writeWithMemFD(p)
	}
	return c.unixConn.Write(p)
}

func (c *JournalConn) writeWithMemFD(p []byte) (n int, err error) {
	var file *os.File
	file, err = ioutil.TempFile("/dev/shm/", "journal.XXXXX")
	if err != nil {
		return
	}
	defer file.Close()

	err = syscall.Unlink(file.Name())
	if err != nil {
		return
	}

	var written int64
	written, err = io.Copy(file, bytes.NewBuffer(p))
	if err != nil {
		return
	}

	rights := syscall.UnixRights(int(file.Fd()))
	// WriteMsgUnix rejects connected datagram sockets, so send the rights
	// through RawConn while retaining net's write readiness handling.
	var raw syscall.RawConn
	raw, err = c.unixConn.SyscallConn()
	if err != nil {
		return
	}
	var sendErr error
	err = raw.Write(func(fd uintptr) bool {
		_, sendErr = syscall.SendmsgN(int(fd), nil, rights, nil, 0)
		return sendErr != syscall.EAGAIN && sendErr != syscall.EWOULDBLOCK
	})
	if err == nil {
		err = sendErr
	}
	if err != nil {
		return
	}

	n = int(written)
	return
}

func (c *JournalConn) Sync() error {
	return nil
}

func (c *JournalConn) Close() error {
	return c.unixConn.Close()
}
