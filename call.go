package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var callData struct {
	lastSuperHeader rewindSuperHeader

	fileName string
	outFile  *os.File

	startedAt         time.Time
	lastFrameReceived time.Time

	ongoing bool

	stdinPipe      io.WriteCloser
	cmd1           *exec.Cmd
	cmd1stderrPipe io.ReadCloser
	cmd2           *exec.Cmd
	cmd2stderrPipe io.ReadCloser
	cmd3           *exec.Cmd
	cmd3stderrPipe io.ReadCloser
}

func createCallFile() {
	outdir := filepath.Clean(settings.OutputDir) + "/" + fmt.Sprintf("%d/%d/%02d/%02d",
		callData.lastSuperHeader.DstID, callData.startedAt.Year(), callData.startedAt.Month(),
		callData.startedAt.Day())
	os.MkdirAll(outdir, 0777)
	callData.fileName = outdir + "/" + fmt.Sprintf("%02d%02d%02d-%d.%s",
		callData.startedAt.Hour(), callData.startedAt.Minute(), callData.startedAt.Second(),
		callData.lastSuperHeader.SrcID, settings.OutputFileExtension)
	var err error
	callData.outFile, err = os.Create(callData.fileName)
	if err != nil {
		log.Println("warning: can't create output file: ", callData.outFile)
	}
}

func logCall(event string) {
	callTypeStr := "group"
	if callData.lastSuperHeader.SessionType == rewindSessionTypePrivateVoice {
		callTypeStr = "private"
	}
	log.Printf("%s call %s, dst: %d src: %d (%s)", callTypeStr, event,
		callData.lastSuperHeader.DstID, callData.lastSuperHeader.SrcID,
		callData.lastSuperHeader.SrcCall)
}

func handleCallStart(sh rewindSuperHeader) {
	callData.lastSuperHeader = sh
	callData.startedAt = time.Now()
	callData.lastFrameReceived = time.Now()
	callData.ongoing = true

	logCall("started")

	createCallFile()

	if len(settings.CallExecCommand1) > 0 {
		cp := strings.Split(settings.CallExecCommand1, " ")
		callData.cmd1 = exec.Command(cp[0], cp[1:]...)

		var err error
		callData.stdinPipe, err = callData.cmd1.StdinPipe()
		if err != nil {
			log.Fatal("can't get stdin pipe for exec command 1")
		}
		if settings.CallExecCommand1ShowStderr {
			callData.cmd1stderrPipe, err = callData.cmd1.StderrPipe()
			if err != nil {
				log.Fatal("can't get stderr pipe for exec command 1")
			}
		}

		if len(settings.CallExecCommand2) > 0 {
			cp = strings.Split(settings.CallExecCommand2, " ")
			callData.cmd2 = exec.Command(cp[0], cp[1:]...)

			// Linking cmd1's stdout to cmd2's stdin
			callData.cmd2.Stdin, err = callData.cmd1.StdoutPipe()
			if err != nil {
				log.Fatal("can't get stdin pipe for exec command 2")
			}

			if settings.CallExecCommand2ShowStderr {
				callData.cmd2stderrPipe, err = callData.cmd2.StderrPipe()
				if err != nil {
					log.Fatal("can't get stderr pipe for exec command 2")
				}
			}

			if len(settings.CallExecCommand3) > 0 {
				cp = strings.Split(settings.CallExecCommand3, " ")
				callData.cmd3 = exec.Command(cp[0], cp[1:]...)

				// Linking cmd2's stdout to cmd3's stdin
				callData.cmd3.Stdin, err = callData.cmd2.StdoutPipe()
				if err != nil {
					log.Fatal("can't get stdin pipe for exec command 3")
				}

				callData.cmd3.Stdout = callData.outFile

				if settings.CallExecCommand3ShowStderr {
					callData.cmd3stderrPipe, err = callData.cmd3.StderrPipe()
					if err != nil {
						log.Fatal("can't get stderr pipe for exec command 3")
					}
				}

				err = callData.cmd3.Start()
				if err != nil {
					log.Fatal("can't start command 3")
				}

				if settings.CallExecCommand3ShowStderr {
					go handleCmdErrPipe(callData.cmd3stderrPipe)
				}
			} else {
				callData.cmd2.Stdout = callData.outFile
			}

			err = callData.cmd2.Start()
			if err != nil {
				log.Fatal("can't start command 2")
			}

			if settings.CallExecCommand2ShowStderr {
				go handleCmdErrPipe(callData.cmd2stderrPipe)
			}
		} else {
			callData.cmd1.Stdout = callData.outFile
		}

		err = callData.cmd1.Start()
		if err != nil {
			log.Fatal("can't run command 1")
		}

		if settings.CallExecCommand1ShowStderr {
			go handleCmdErrPipe(callData.cmd1stderrPipe)
		}
	}
}

func handleCmdErrPipe(pipe io.ReadCloser) {
	b := make([]byte, 1024)
	for {
		if pipe == nil {
			break
		}
		c, err := pipe.Read(b)
		if c > 0 {
			log.Println(string(b))
		}
		if err != nil {
			break
		}
	}
}

func handleDMRAudioFrame(payload []byte) {
	if callData.ongoing {
		callData.lastFrameReceived = time.Now()
		if callData.stdinPipe != nil {
			callData.stdinPipe.Write(payload)
		} else {
			callData.outFile.Write(payload)
		}
	}
}

func handleCallEnd() {
	if !callData.ongoing {
		return
	}

	logCall("ended")

	if callData.cmd1 != nil {
		callData.stdinPipe.Close()
		callData.stdinPipe = nil

		if callData.cmd1stderrPipe != nil {
			callData.cmd1stderrPipe.Close()
			callData.cmd1stderrPipe = nil
		}
		callData.cmd1.Wait()
		callData.cmd1 = nil

		if callData.cmd2 != nil {
			if callData.cmd2stderrPipe != nil {
				callData.cmd2stderrPipe.Close()
				callData.cmd2stderrPipe = nil
			}
			callData.cmd2.Wait()
			callData.cmd2 = nil

			if callData.cmd3 != nil {
				if callData.cmd3stderrPipe != nil {
					callData.cmd3stderrPipe.Close()
					callData.cmd3stderrPipe = nil
				}
				callData.cmd3.Wait()
				callData.cmd3 = nil
			}
		}
	}

	if callData.outFile != nil {
		callData.outFile.Close()
		callData.outFile = nil
	}

	if settings.CreateDailyAggregateFile {
		fi, err := os.Open(callData.fileName)
		if err != nil {
			log.Println("warning: can't open call rec file to append to daily aggregate")
		} else {
			defer fi.Close()

			filename := fmt.Sprintf("%s/%d%02d%02d-%d.%s", filepath.Dir(callData.fileName),
				callData.startedAt.Year(), callData.startedAt.Month(), callData.startedAt.Day(),
				callData.lastSuperHeader.DstID, settings.OutputFileExtension)
			fo, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				log.Println("warning: can't append to daily aggregate", filename)
			} else {
				defer fo.Close()

				b := make([]byte, 1024)
				for {
					c, err := fi.Read(b)
					if c == 0 || err != nil {
						break
					}
					fo.Write(b[:c])
				}
			}
		}
	}

	callData.ongoing = false
}
