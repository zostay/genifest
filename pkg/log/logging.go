package log

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kr/pretty"

	cfgstr "github.com/zostay/genifest/pkg/strtools"
)

const (
	// File is the default location of the current log file.
	File = "log/genifest.log"
)

var (
	LogCloser io.Closer     // provides a closer when needed
	logger    io.Writer     // log entries are written here
	memLogger *bytes.Buffer // this buffer keeps an in-memory version of the logs
)

// Setup rotates the log files if the first line is from a different day,
// then opens up the current log file for append.
func Setup(cloudHome, logPath string, useStderr, forceRotate bool) error {
	if useStderr {
		logger = os.Stderr
	} else {
		f, err := RotateAndOpenLogfile(cloudHome, logPath, forceRotate)
		if err != nil {
			return err
		}

		logger = f
		LogCloser = f
	}

	// restart memLogger with each call
	memLogger = new(bytes.Buffer)
	logger = io.MultiWriter(logger, memLogger)

	return nil
}

// used to match the date on the first line of the log file to test for the need
// to rotate.
var dateMatch = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)

// RotateAndOpenLogfile tests to see if the first line of the log is from a
// different day than today. If so, it will rotate the log file by appending the
// date of the first entry to the end ofthe log filename and renaming the file
// to that. It will then open the current log file for append, creating a new
// log file if one does not exist following rotation.
func RotateAndOpenLogfile(cloudHome, logPath string, force bool) (io.WriteCloser, error) {
	if logPath == "" {
		logPath = File
	}

	logFile := filepath.Join(cloudHome, logPath)
	if lfi, err := os.Stat(logFile); err == nil && !lfi.IsDir() {
		r, err := os.Open(logFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file for reading: %w", err)
		}

		lines := bufio.NewScanner(r)
		var firstLine string
		if lines.Scan() {
			firstLine = lines.Text()
		}

		if dms := dateMatch.FindStringSubmatch(firstLine); force || dms != nil {
			dateStr := dms[1]
			nowStr := time.Now().Format("2006-01-02")
			if force || dateStr != nowStr {
				arcLogFile := logFile + "." + dateStr

				// make sure we don't clobber an existing, which can happen
				// particularly when rotation is being forced
				i := -1
				for {
					testFile := arcLogFile
					if i >= 0 {
						testFile = fmt.Sprintf("%s.%03d", testFile, i)
					}

					if _, err := os.Stat(testFile); os.IsNotExist(err) {
						arcLogFile = testFile
						break
					}

					i++
				}

				fmt.Printf("Rotating old %q to %q\n", logFile, arcLogFile)
				err := os.Rename(logFile, arcLogFile)
				if err != nil {
					return nil, fmt.Errorf("unable to rename file to rotate: %w", err)
				}
			}
		}
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file for writing: %w", err)
	}
	return f, nil
}

// Line records a log message with the given prefix.
func Line(prefix, msg string) {
	ts := time.Now().Format("[2006-01-02T15:04:05.000000-07:00]")
	fmt.Fprintf(logger, "%s %s %s\n", ts, prefix, cfgstr.IndentSpaces(len(ts)+len(prefix)+2, msg))
}

// LineAndSay records a log message with the given prefix and write the message out
// to stdout as well.
func LineAndSay(prefix, msg string) {
	Line(prefix, msg)
	fmt.Printf("\n%s %s\n", prefix, cfgstr.IndentSpaces(len(prefix)+1, msg))
}

// LineBytes records a log message from a byte slice.
func LineBytes(prefix string, msg []byte) {
	Line(prefix, string(msg))
}

// LineStream records a log message from a stream of data as it arrives.
func LineStream(prefix string, msg io.Reader) {
	r := bufio.NewReader(msg)
	go func() {
		var err error
		for {
			var line []byte
			var pre bool
			line, pre, err = r.ReadLine()
			if err != nil {
				break
			}

			if pre {
				line = append(line, []byte("…")...)
			} else {
				line = bytes.TrimSpace(line)
				line = append(line, []byte("␤")...)
			}

			LineBytes(prefix, line)
		}

		if err != io.EOF {
			Linef("FAIL", "Stream closed with unexpected err: %v", err)
		}
	}()
}

// Linef records a log message using printf-style formatting.
func Linef(prefix, format string, args ...interface{}) {
	msg := pretty.Sprintf(format, args...)
	Line(prefix, msg)
}

// LineAndSayf records a log message and outputs the message to stdout as
// well using printf-style formatting.
func LineAndSayf(prefix, format string, args ...interface{}) {
	msg := pretty.Sprintf(format, args...)
	LineAndSay(prefix, msg)
}
