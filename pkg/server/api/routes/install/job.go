package install

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const (
	routierBin = "/usr/bin/routier"
	logPath    = "/var/log/routier-install.log"
	logLines   = 400
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

type job struct {
	mu    sync.Mutex
	state string
	disk  string
	err   string
	lines []string
}

var current = &job{state: "idle"}

func (j *job) snapshot() types.InstallStatus {
	j.mu.Lock()
	defer j.mu.Unlock()

	lines := make([]string, len(j.lines))
	copy(lines, j.lines)
	return types.InstallStatus{State: j.state, Disk: j.disk, Error: j.err, Log: lines}
}

func (j *job) begin(disk string) bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.state == "running" || j.state == "done" {
		return false
	}

	j.state, j.disk, j.err, j.lines = "running", disk, "", nil
	return true
}

func (j *job) finish(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if err != nil {
		j.state, j.err = "failed", err.Error()
		return
	}

	j.state = "done"
}

func (j *job) append(line string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.lines = append(j.lines, line)
	if len(j.lines) > logLines {
		j.lines = j.lines[len(j.lines)-logLines:]
	}
}

func (j *job) run(req types.InstallRequest) {
	cmd := exec.Command(routierBin, "setup",
		"--disk", req.Disk,
		"--hostname", req.Hostname,
		"--timezone", req.Timezone,
		"--password-stdin",
		"--no-reboot",
	)
	cmd.Stdin = strings.NewReader(req.Password + "\n")

	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Warn().Err(err).Msg("install log file not written")
	}

	done := make(chan struct{})
	go j.collect(reader, logFile, done)

	if err := cmd.Start(); err != nil {
		_ = writer.Close()
		<-done
		j.finish(err)
		return
	}

	err = cmd.Wait()
	_ = writer.Close()
	<-done
	if logFile != nil {
		_ = logFile.Close()
	}

	j.finish(err)
	log.Info().Str("disk", req.Disk).AnErr("error", err).Msg("disk installation finished")
}

func (j *job) collect(r io.Reader, logFile *os.File, done chan<- struct{}) {
	defer close(done)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := ansiEscape.ReplaceAllString(scanner.Text(), "")
		if logFile != nil {
			_, _ = logFile.WriteString(line + "\n")
		}

		if strings.TrimSpace(line) != "" {
			j.append(line)
		}
	}
}
