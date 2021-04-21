# callrec

Records BrandMeister calls on a talkgroup to files using the Simple External
Application protocol.

## Installation

```
go get github.com/BrandMeister/callrec
go install github.com/BrandMeister/callrec
```

## Configuration

Copy *config-example.json* to *config.json* and edit it.

You can define `callExecCommand`s. Received call data will be passed to
`callExecCommand1`'s stdin, and call data passed through all defined
`callExecCommand`s will be saved on the disk (`callExecCommand1`'s stdout is piped
to `callExecCommand2`'s stdin and so on). This way you can pass AMBE data
to an external application which decodes the stream.

The strings `$SRCID`, `$DSTID`, `$SRCCALL` and `$DSTCALL` in the
`callExecCommand`s get replaced by their value at the time of the call.

Note that every time a new call starts, a new process gets created. When the call ends,
the processes started by `callExecCommand` get their stdin closed and need to stop.

In some cases (for example when UDP hole punching is not enabled on the firewall), it
might be useful to have a fixed listening port instead of randomly picking one. You can
do this by setting `sourceAddress` to for example `:1234`: then port 1234 will be used
as the local listening port.
