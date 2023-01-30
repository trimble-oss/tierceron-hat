package crown

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

var cryptMap map[string]string = map[string]string{}

func Tap(target string) error {
	listener, err := net.Listen("unix", "./snap.sock")
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			if conn != nil {
				conn.Close()
			}
			return err
		}

		// 1st check.
		if conn.RemoteAddr().Network() == conn.LocalAddr().Network() {

			// 2nd check.
			sysConn, sysConnErr := conn.(*net.UnixConn).SyscallConn()
			if sysConnErr != nil {
				conn.Close()
				continue
			}

			var cred *unix.Ucred

			sysConn.Control(func(fd uintptr) {
				cred, err = unix.GetsockoptUcred(int(fd),
					unix.SOL_SOCKET,
					unix.SO_PEERCRED)
			})

			path, linkErr := os.Readlink("/proc/" + strconv.Itoa(int(cred.Pid)) + "/exe")
			if linkErr != nil {
				conn.Close()
				continue
			}
			defer conn.Close()

			if path == target {
				go func(c net.Conn) {

					buff := &bytes.Buffer{}
					io.Copy(conn, buff)
					if buff.Len() == 32 {
						cryptMap[buff.String()] = ""
					}
					buff.Reset()

				}(conn)
			}

		}
	}

}

func main() {
	ex, err := os.Executable()
	if err != nil {
		os.Exit(-1)
	}
	exPath := filepath.Dir(ex)
	brimPath := strings.Replace(exPath, "/crown", "/brim", 1)
	Tap(brimPath)
}
