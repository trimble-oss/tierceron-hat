//go:build linux
// +build linux

package tap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

const penseSocket = "./snap.sock"

func Tap(penseDir string, tapMap map[string]string, group string, skipPathControls bool) error {
	// Tap always starts with a clean slate.

	var penseDirSocket = filepath.Clean(penseDir + penseSocket)

	err := os.MkdirAll(penseDir, 0770)
	if err != nil {
		return errors.Join(errors.New("dir create error"), err)
	}
	azureDeployGroup, azureDeployGroupErr := user.LookupGroup(group)
	if azureDeployGroupErr != nil {
		return errors.Join(errors.New("group lookup failure"), azureDeployGroupErr)
	}
	azureDeployGID, azureGIDConvErr := strconv.Atoi(azureDeployGroup.Gid)
	if azureGIDConvErr != nil {
		return errors.Join(errors.New("group ID lookup failure"), azureGIDConvErr)
	}
	os.Chown(penseDir, -1, azureDeployGID)
	os.Chmod(penseDir, 0770)
	os.Remove(penseDirSocket)
	origUmask := syscall.Umask(0777)
	listener, listenErr := net.Listen("unix", penseDirSocket)
	syscall.Umask(origUmask)
	os.Chown(penseDirSocket, -1, azureDeployGID)
	os.Chmod(penseDirSocket, 0770)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGABRT)

	go func(c chan os.Signal) {
		<-c
		if listener != nil {
			listener.Close()
		}
		os.Remove(penseDirSocket)
		os.Exit(0)
	}(signalChan)

	if err != nil {
		return errors.Join(errors.New("listen error"), listenErr)
	}

	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			if conn != nil {
				conn.Close()
			}
			return errors.Join(errors.New("accept error"), acceptErr)
		}

		// 1st check.
		if conn.RemoteAddr().Network() == conn.LocalAddr().Network() {
			sysConn, sysConnErr := conn.(*net.UnixConn).SyscallConn()
			if !skipPathControls && sysConnErr != nil {
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
			if !skipPathControls && credErr != nil {
				conn.Close()
				continue
			}

			path, linkErr := os.Readlink("/proc/" + strconv.Itoa(int(cred.Pid)) + "/exe")

			if !skipPathControls && linkErr != nil {
				conn.Close()
				continue
			}

			// 2nd check.
			if expectedSha256, ok := tapMap[path]; skipPathControls || ok {
				// 3rd check.
				peerExe, err := os.Open(path)
				if !skipPathControls && err != nil {
					conn.Close()
					continue
				}
				// Close in the current scope rather than defer
				h := sha256.New()
				if _, err := io.Copy(h, peerExe); !skipPathControls && err != nil {
					peerExe.Close()
					conn.Close()
					continue
				}
				peerExe.Close()

				if skipPathControls || expectedSha256 == hex.EncodeToString(h.Sum(nil)) {
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
							conn.Close()
							continue
						}
						_, writeErr := conn.Write([]byte(eyes))
						if writeErr != nil {
							conn.Close()
							continue
						}
					}
				}
			}
		}
		conn.Close()
	}
}

func TapWriter(penseDir string, pense string) (map[string]string, error) {
	var penseDirSocket = filepath.Clean(penseDir + penseSocket)
	penseConn, penseErr := net.Dial("unix", penseDirSocket)
	if penseErr != nil {
		return nil, penseErr
	}
	defer penseConn.Close()

	_, penseWriteErr := penseConn.Write([]byte(pense))
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
