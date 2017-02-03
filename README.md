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

You can define callExecCommands. Received call data will be passed to
callExecCommand1's stdin, and call data passed through all defined
callExecCommands will be saved on the disk (callExecCommand1's stdout is piped
to callExecCommand2's stdin and so on). This way you can pass AMBE data
to an external application which decodes the stream.
