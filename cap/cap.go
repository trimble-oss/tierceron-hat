package cap

import (
	context "context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	grpc "google.golang.org/grpc"
)

var penseCodeMap map[string]string = map[string]string{}
var penseMemoryMap map[string]string = map[string]string{}

const penseSocket = "trcsnap.sock"
const penseDir = "/tmp/trccarrier/"

func TapServer(address string, opt ...grpc.ServerOption) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var s *grpc.Server
	if opt != nil {
		s = grpc.NewServer(opt...)
	} else {
		s = grpc.NewServer()
	}
	RegisterCapServer(s, &penseServer{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

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
					conn.Close()
					continue
				}

				if skipPathControls || expectedSha256 == hex.EncodeToString(h.Sum(nil)) {
					messageBytes := make([]byte, 64)

					err := sysConn.Read(func(s uintptr) bool {
						_, operr := syscall.Read(int(s), messageBytes)
						return operr != syscall.EAGAIN
					})
					if err != nil {
						conn.Close()
						continue
					}
					message := string(messageBytes)

					if len(message) == 64 {
						penseCodeMap[message] = ""
					}
				}

			}

		}
		conn.Close()
	}
}

func TapWriter(pense string) error {
	penseConn, penseErr := net.Dial("unix", penseDir+penseSocket)
	if penseErr != nil {
		return penseErr
	}
	_, penseWriteErr := penseConn.Write([]byte(pense))
	defer penseConn.Close()
	if penseWriteErr != nil {
		return penseWriteErr
	}

	_, penseResponseErr := io.ReadAll(penseConn)

	return penseResponseErr
}

func TapMemorize(penseIndex, memory string) {
	penseMemoryMap[penseIndex] = memory
}

type penseServer struct {
	UnimplementedCapServer
}

func (cs *penseServer) Pense(ctx context.Context, penseRequest *PenseRequest) (*PenseReply, error) {

	penseArray := sha256.Sum256([]byte(penseRequest.Pense))
	penseCode := hex.EncodeToString(penseArray[:])
	if _, penseCodeOk := penseCodeMap[penseCode]; penseCodeOk {
		delete(penseCodeMap, penseCode)

		if pense, penseOk := penseMemoryMap[penseRequest.PenseIndex]; penseOk {
			return &PenseReply{Pense: pense}, nil
		} else {
			return &PenseReply{Pense: "Pense undefined"}, nil
		}
	} else {
		return &PenseReply{Pense: "...."}, nil
	}
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		os.Exit(-1)
	}
	exePath := filepath.Dir(ex)
	brimPath := strings.Replace(exePath, "/Cap", "/brim", 1)
	go Tap(brimPath, "f19431f322ea015ef871d267cc75e58b73d16617f9ff47ed7e0f0c1dbfb276b5", "", false)
	TapServer("127.0.0.1:1534")

}
