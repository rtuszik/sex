// Package termbg asks the terminal for its default background color (OSC 11).
package termbg

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var rgbRe = regexp.MustCompile(`rgb:([0-9a-fA-F]{1,4})/([0-9a-fA-F]{1,4})/([0-9a-fA-F]{1,4})`)

// Parse turns an OSC 11 reply such as "rgb:1616/1616/1616" into "#161616".
func Parse(reply string) (string, bool) {
	m := rgbRe.FindStringSubmatch(reply)
	if m == nil {
		return "", false
	}
	var rgb [3]int
	for i, hex := range m[1:] {
		v, err := strconv.ParseUint(hex, 16, 16)
		if err != nil {
			return "", false
		}
		limit := math.Pow(16, float64(len(hex))) - 1
		rgb[i] = int(math.Round(float64(v) / limit * 255))
	}
	return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2]), true
}

// Query writes the OSC 11 query to /dev/tty and waits up to 200ms for the reply.
func Query() (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", err
	}
	defer func() { _ = tty.Close() }()
	saved, err := stty(tty, "-g")
	if err != nil {
		return "", err
	}
	defer func() { _, _ = stty(tty, saved) }()
	if _, err := stty(tty, "raw", "-echo", "min", "0", "time", "2"); err != nil {
		return "", err
	}
	if _, err := tty.WriteString("\x1b]11;?\x1b\\"); err != nil {
		return "", err
	}
	var reply []byte
	buf := make([]byte, 64)
	for len(reply) < 256 {
		n, err := tty.Read(buf)
		if n == 0 || err != nil {
			break
		}
		reply = append(reply, buf[:n]...)
		if bytes.HasSuffix(reply, []byte("\a")) || bytes.HasSuffix(reply, []byte("\x1b\\")) {
			break
		}
	}
	if color, ok := Parse(string(reply)); ok {
		return color, nil
	}
	return "", errors.New("terminal did not report its background")
}

func stty(tty *os.File, args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = tty
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
