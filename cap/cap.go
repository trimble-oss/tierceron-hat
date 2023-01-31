package cap

import (
	"bytes"
	context "context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
	grpc "google.golang.org/grpc"
)

var penseCodeMap map[string]string = map[string]string{}
var penseMap map[string]string = map[string]string{}

const penseSocket = "./snap.sock"

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

func Tap(target string, expectedSha256 string) error {
	listener, err := net.Listen("unix", penseSocket)
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

			// 2nd check.
			if path == target {

				// 3rd check.
				targetFile, err := os.Open(path)
				if err != nil {
					continue
				}
				defer targetFile.Close()

				h := sha256.New()
				if _, err := io.Copy(h, targetFile); err != nil {
					continue
				}

				if expectedSha256 == hex.EncodeToString(h.Sum(nil)) {
					go func(c net.Conn) {
						buff := &bytes.Buffer{}
						io.Copy(conn, buff)
						if buff.Len() == 32 {
							penseCodeMap[buff.String()] = ""
						}
						buff.Reset()

					}(conn)
				}
			}

		}
	}
}

func TapWriter(pense string) error {
	penseConn, penseErr := net.Dial("unix", penseSocket)
	if penseErr != nil {
		return penseErr
	}
	_, penseWriteErr := penseConn.Write([]byte(pense))
	if penseWriteErr != nil {
		return penseWriteErr
	}

	_, penseResponseErr := io.ReadAll(penseConn)

	return penseResponseErr
}

type penseServer struct {
	UnimplementedCapServer
}

func (cs *penseServer) Pense(ctx context.Context, penseRequest *PenseRequest) (*PenseReply, error) {

	penseArray := sha256.Sum256([]byte(penseRequest.Pense))
	penseCode := hex.EncodeToString(penseArray[:])
	if _, penseCodeOk := penseCodeMap[penseCode]; penseCodeOk {

		if pense, penseOk := penseMap[penseRequest.PenseIndex]; penseOk {
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
	go Tap(brimPath, "f19431f322ea015ef871d267cc75e58b73d16617f9ff47ed7e0f0c1dbfb276b5")
	TapServer("127.0.0.1:1534")

}
