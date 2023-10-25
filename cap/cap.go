package cap

import (
	context "context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
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
	"time"

	"errors"

	"github.com/lafriks/go-shamir"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/sys/unix"
	grpc "google.golang.org/grpc"
)

const (
	FEATHER_COMMON = 1 << iota // COMMON
	FEATHER_CTL    = 1 << iota // CTL 2
	FEATHER_SECRET = 1 << iota // SECRET 4
)

const (
	MODE_PERCH = "c"
	MODE_FLAP  = "p"
	MODE_GLIDE = "g"
	MODE_GAZE  = "z"
)

var penseCodeMap map[string]string = map[string]string{}
var penseMemoryMap map[string]string = map[string]string{}

var penseFeatherCodeMap map[string]string = map[string]string{}
var penseFeatherMemoryMap map[string]string = map[string]string{}

var penseFeatherCtlCodeMap = cmap.New[string]()

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

var clientCodeMap map[string][][]byte = map[string][][]byte{}

func handleMessage(handshakeCode string, conn *kcp.UDPSession, acceptRemote func(int, string) bool) {
	buf := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if _, ok := clientCodeMap[conn.RemoteAddr().String()]; !ok {
			clientCodeMap[conn.RemoteAddr().String()] = [][]byte{}
		}

		if n == 0 || err != nil {
			// All done... hopefully.
			if _, ok := clientCodeMap[conn.RemoteAddr().String()]; ok {
				var messageBytes []byte
				var err error = nil
				if len(clientCodeMap[conn.RemoteAddr().String()]) > 1 {
					messageBytes, err = shamir.Combine(clientCodeMap[conn.RemoteAddr().String()]...)
				} else {
					if acceptRemote(FEATHER_CTL, conn.RemoteAddr().String()) {
						messageBytes = clientCodeMap[conn.RemoteAddr().String()][0]
						clientCodeMap[conn.RemoteAddr().String()][0] = []byte{}
						message := string(messageBytes)
						messageParts := strings.Split(message, ":")
						if messageParts[0] == handshakeCode {
							// handshake:featherctl:f|p|g:activity
							if messageParts[1] == "featherctl" && len(messageParts) == 4 {
								var msg string = ""
								var ok bool
								if msg, ok = penseFeatherCtlCodeMap.Get(messageParts[3]); !ok {
									// Default is Glide
									msg = MODE_GLIDE
								}
								if len(messageParts[3]) < 20 && len(messageParts[2]) < 100 {
									switch {
									case strings.HasPrefix(messageParts[2], MODE_PERCH): // Perch
										penseFeatherCtlCodeMap.Set(messageParts[3], messageParts[2])
									case strings.HasPrefix(messageParts[2], MODE_FLAP): // Flap
										penseFeatherCtlCodeMap.Set(messageParts[3], messageParts[2])
									case strings.HasPrefix(messageParts[2], MODE_GLIDE): // Glide
										penseFeatherCtlCodeMap.Set(messageParts[3], messageParts[2])
									}
								}
								conn.Write([]byte(msg))
								defer conn.Close()
								return
							}
						}
					}
				}
				if err == nil {
					if acceptRemote(FEATHER_SECRET, conn.RemoteAddr().String()) {
						message := string(messageBytes)
						messageParts := strings.Split(message, ":")
						if messageParts[0] == handshakeCode {
							if len(messageParts[1]) == 64 {
								penseFeatherCodeMap[messageParts[1]] = ""
							}
						}
					}
				}
			}
			conn.Write([]byte(" "))
			defer conn.Close()
			return
		} else {
			clientCodeMap[conn.RemoteAddr().String()] = append(clientCodeMap[conn.RemoteAddr().String()], append([]byte{}, buf[:n]...))
		}
	}
}

func Feather(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, acceptRemote func(int, string) bool) {
	key := pbkdf2.Key([]byte(encryptPass), []byte(encryptSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	if listener, err := kcp.ListenWithOptions(hostAddr, block, 10, 3); err == nil {
		for {
			s, err := listener.AcceptKCP()
			if err != nil {
				log.Fatal(err)
			}
			if acceptRemote(FEATHER_COMMON, s.RemoteAddr().String()) {
				go handleMessage(handshakeCode, s, acceptRemote)
			} else {
				s.Close()
			}
		}
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

func FeatherCtlEmit(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, modeCtlPack string, pense string) (string, error) {
	key := pbkdf2.Key([]byte(encryptPass), []byte(encryptSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	penseConn, penseErr := kcp.DialWithOptions(hostAddr, block, 10, 3)
	if penseErr != nil {
		return "", penseErr
	}
	defer penseConn.Close()
	_, penseWriteErr := penseConn.Write([]byte(handshakeCode + ":featherctl:" + modeCtlPack + ":" + pense))
	if penseWriteErr != nil {
		return "", penseWriteErr
	}

	responseBuf := []byte{1}
	_, penseResponseErr := io.ReadFull(penseConn, responseBuf)

	return string(responseBuf), penseResponseErr
}

func FeatherWriter(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, pense string) ([]byte, error) {
	penseSplits, err := shamir.Split([]byte(handshakeCode+":"+pense), 12, 7)
	if err != nil {
		return nil, err
	}
	key := pbkdf2.Key([]byte(encryptPass), []byte(encryptSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	penseConn, penseErr := kcp.DialWithOptions(hostAddr, block, 10, 3)
	if penseErr != nil {
		return nil, penseErr
	}
	defer penseConn.Close()
	for _, penseBlock := range penseSplits {
		_, penseWriteErr := penseConn.Write(penseBlock)
		if penseWriteErr != nil {
			return nil, penseWriteErr
		}
	}

	responseBuf := []byte{1}
	_, penseResponseErr := io.ReadFull(penseConn, responseBuf)

	return responseBuf, penseResponseErr
}

func TapFeather(penseIndex, memory string) {
	penseMemoryMap[penseIndex] = memory
	penseFeatherMemoryMap[penseIndex] = memory
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
		// Might be a feather
		if _, penseCodeOk := penseFeatherCodeMap[penseCode]; penseCodeOk {
			delete(penseFeatherCodeMap, penseCode)
			if pense, penseOk := penseFeatherMemoryMap[penseRequest.PenseIndex]; penseOk {
				return &PenseReply{Pense: pense}, nil
			} else {
				return &PenseReply{Pense: "Pense undefined"}, nil
			}
		}
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
