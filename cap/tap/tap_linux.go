//go:build linux
// +build linux

package tap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

const penseSocket = "./snap.sock"

func Tap(target string, expectedSha256 string) error {
	listener, err := net.Listen("unix", penseSocket)
	if err != nil {
		return err
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func(c chan os.Signal) {
		<-c
		listener.Close()
		os.Exit(0)
	}(signalChan)

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

			sysConn, sysConnErr := conn.(*net.UnixConn).SyscallConn()
			if sysConnErr != nil {
				conn.Close()
				continue
			}

			var cred *unix.Ucred
			var credErr error

			sysConn.Control(func(fd uintptr) {
				cred, credErr = unix.GetsockoptUcred(int(fd),
					unix.SOL_SOCKET,
					unix.SO_PEERCRED)
			})
			if credErr != nil {
				conn.Close()
				continue
			}

			path, linkErr := os.Readlink("/proc/" + strconv.Itoa(int(cred.Pid)) + "/exe")
			if linkErr != nil {
				conn.Close()
				continue
			}
			defer conn.Close()

			// 2nd check.
			if path == target {
				// 3rd check.
				peerExe, err := os.Open(path)
				if err != nil {
					conn.Close()
					continue
				}

				h := sha256.New()
				if _, err := io.Copy(h, peerExe); err != nil {
					peerExe.Close()
					conn.Close()
					continue
				}
				peerExe.Close()

				if expectedSha256 == hex.EncodeToString(h.Sum(nil)) {
					messageBytes := make([]byte, 64)
					_, err := conn.Read(messageBytes)
					if err != nil {
						conn.Close()
						continue
					}
					message := string(messageBytes)

					if len(message) == 64 {
						penseCodeMap[message] = ""
						eyes, err := json.Marshal(penseEyeMap)
						if err != nil {
							conn.Write([]byte("mad eye"))
						}
						conn.Write([]byte(eyes))
					}
				}
			}
		}
		conn.Close()
	}
}

func TapWriter(pense string) (map[string]string, error) {
	penseConn, penseErr := net.Dial("unix", penseSocket)
	if penseErr != nil {
		return nil, penseErr
	}
	_, penseWriteErr := penseConn.Write([]byte(pense))
	defer penseConn.Close()
	if penseWriteErr != nil {
		return nil, penseWriteErr
	}
	eyeMapRaw, penseResponseErr := io.ReadAll(penseConn)

	if penseResponseErr == nil {
		eyeMap := map[string]string{}
		penseResponseDeserializeErr := json.Unmarshal(eyeMapRaw, &eyeMap)
		return eyeMap, penseResponseDeserializeErr
	}

	return nil, penseResponseErr
}
