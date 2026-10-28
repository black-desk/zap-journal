<!--
SPDX-FileCopyrightText: 2023-2026 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# zap-journal
This is a extension of zap
which implement `zapcore.Encoder` and `zapcore.WriteSyncer` of systemd-journal.

Check ./example/example.go for usage.

## Testing

Run the tests with:

```sh
go test -count=1 -v ./...
```

Run only the integration tests with:

```sh
go test -count=1 -v -timeout=1m ./test/integration
```

`TestJournalIntegration` in `test/integration/` uses `helper.sh` to start a real
systemd-journald in isolated user, mount, and PID namespaces. The Go test stays
outside those namespaces, connects through a shared temporary socket directory,
and reads the shared journal directory with journalctl. It checks logger
initialization failures and writes structured logs through zap, the journal
encoder, and the journal connection. Both ordinary datagrams and large entries
sent by file descriptor are verified, including multiline field contents. It
does not use or modify the host journal.

The integration test requires `unshare`, `sh`, `mount`, `systemd-journald`, and
`journalctl`, plus permission to create user namespaces and private mounts. It
runs `helper.sh --check` before startup to locate the systemd tools and check
namespace and mount support; it skips if that check fails. The script's
`--read` mode invokes journalctl outside the namespaces. All
subsequent startup, connection, write, and readback errors fail the test.
Waiting is bounded only by `go test -timeout`; choose a timeout suitable for
the local machine or CI. Closing the test process's stdin pipe tells the helper
to stop journald and remove its temporary directory after the namespace exits,
including when a test timeout prevents Go cleanup from running.
Use `-v` to see skip reasons and `-count=1` to rerun without cached results.

## Thanks

- [go-systemd](https://github.com/coreos/go-systemd)
- [zapext](https://github.com/tchap/zapext)

## Note

1. The default logger setup (`zapjournal.New()` and `zapjournal.NewDebug()`)
   produce a named logger with the executable file name
   get from `os.Executable()`.
   This behavior is different from zap.

2. The key passed to zap will be convert to **UPPERCASE**.
   And all ` ` in key will be replaced with `_`.
   As journald seems refuse to record such fields.

3. `journalctl` will not show any custom fields by default.
   And it not support any format customization option to display custom fields.
   But it has an option to output log as json which can be pass to `jq`,
   Then that be formatted to a human readable output.

   Check [journalfmt][1] as well.

[1]: https://github.com/black-desk/journalfmt

## License

This project is licensed under the [MIT License](LICENSES/MIT.txt).
Code copied from zap retains Uber Technologies, Inc.'s copyright and MIT notice.
