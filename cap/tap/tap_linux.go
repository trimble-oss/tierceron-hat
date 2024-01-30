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
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

const penseSocket = "./snap.sock"
const penseDir = "/tmp/trccarrier/"

func Tap(target string, expectedSha256 string, group string, skipPathControls bool) error {
	// Tap always starts with a clean slate.
	err := os.MkdirAll(penseDir, 0770)
	if err != nil {
		return errors.Join(errors.New("Dir create error"), err)
	}
	azureDeployGroup, azureDeployGroupErr := user.LookupGroup(group)
	if azureDeployGroupErr != nil {
		return errors.Join(errors.New("Group lookup failure"), azureDeployGroupErr)
	}
	azureDeployGID, azureGIDConvErr := strconv.Atoi(azureDeployGroup.Gid)
	if azureGIDConvErr != nil {
		return errors.Join(errors.New("Group ID lookup failure"), azureGIDConvErr)
	}
	os.Chown(penseDir, -1, azureDeployGID)
	os.Chmod(penseDir, 0770)
	os.Remove(penseDir + penseSocket)
	origUmask := syscall.Umask(0777)
	listener, listenErr := net.Listen("unix", penseDir+penseSocket)
	syscall.Umask(origUmask)
	os.Chown(penseDir+penseSocket, -1, azureDeployGID)
	os.Chmod(penseDir+penseSocket, 0770)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGABRT)

	go func(c chan os.Signal) {
		<-c
		if listener != nil {
			listener.Close()
		}
		os.Remove(penseDir + penseSocket)
		os.Exit(0)
	}(signalChan)

	if err != nil {
		return errors.Join(errors.New("Listen error"), listenErr)
	}

	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			if conn != nil {
				conn.Close()
			}
			return errors.Join(errors.New("Accept error"), acceptErr)
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
			if skipPathControls || path == target {
				// 3rd check.
				peerExe, err := os.Open(path)
				if !skipPathControls && err != nil {
					conn.Close()
					continue
				}
				defer peerExe.Close()

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
	penseConn, penseErr := net.Dial("unix", penseDir+penseSocket)
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
