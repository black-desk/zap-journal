# zap-journal
This is a extension of zap
which implement `zapcore.Encoder` and `zapcore.WriteSyncer` of systemd-journal.

Check ./example/example.go for usage.

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
