package cap

import (
	context "context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap/tap"

	"github.com/lafriks/go-shamir"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/pbkdf2"
	grpc "google.golang.org/grpc"
)

const (
	FEATHER_COMMON = 1 << iota // COMMON
	FEATHER_CTL    = 1 << iota // CTL 2
	FEATHER_SECRET = 1 << iota // SECRET 4
)

const (
	MODE_PERCH = "c"
	MOID_VOID  = "v"
	MODE_PLUCK = "k"
	MODE_FLAP  = "p"
	MODE_GLIDE = "g"
	MODE_GAZE  = "z"
)

var penseMemoryMap map[string]string = map[string]string{}

var penseFeatherCodeMap = cmap.New[string]()
var penseFeatherMemoryMap map[string]string = map[string]string{}

var penseFeatherPluckMap = cmap.New[bool]()
var penseFeatherCtlCodeMap = cmap.New[string]()

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

var clientCodeMap = cmap.New[[][]byte]()

func handlePluck(conn *kcp.UDPSession, acceptRemote func(int, string) bool) {
	buf := make([]byte, 50)
	for {
		conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		message := string(buf[:n])
		messageParts := strings.Split(message, ":")

		if messageParts[0] == MODE_PLUCK {
			if len(messageParts[1]) > 0 {
				if _, ok := penseFeatherPluckMap.Pop(messageParts[1]); ok {
					conn.Write([]byte(MODE_PLUCK))
					defer conn.Close()
					return
				} else {
					conn.Write([]byte(MOID_VOID))
					defer conn.Close()
					return
				}
			}
		} else {
			return
		}
	}
}

func handleMessage(handshakeCode string, conn *kcp.UDPSession, acceptRemote func(int, string) bool) {
	buf := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if _, ok := clientCodeMap.Get(conn.RemoteAddr().String()); !ok {
			clientCodeMap.Set(conn.RemoteAddr().String(), [][]byte{})
		}

		if n == 0 || err != nil {
			// All done... hopefully.
			if _, ok := clientCodeMap.Get(conn.RemoteAddr().String()); ok {
				var messageBytes []byte
				var err error = nil
				if cremote, ok := clientCodeMap.Get(conn.RemoteAddr().String()); ok && len(cremote) > 1 {
					messageBytes, err = shamir.Combine(cremote...)
				} else {
					if acceptRemote(FEATHER_CTL, conn.RemoteAddr().String()) {
						if ok && len(cremote) > 0 {
							messageBytes = cremote[0]
						} else {
							// Race condition... Literally nothing can be done here other than
							// give an empty response and exit.
							goto failover
						}
						cremote[0] = []byte{}
						message := string(messageBytes)

						messageParts := strings.Split(message, ":")
						if messageParts[0] == handshakeCode {
							// handshake:featherctl:f|p|g:activity
							if messageParts[1] == "featherctl" && len(messageParts) == 4 {
								var msg string = ""
								var ok bool
								if msg, ok = penseFeatherCtlCodeMap.Get(messageParts[3]); !ok {
									// Default is Perch
									msg = MODE_PERCH
								}

								if len(messageParts[3]) < 50 && len(messageParts[2]) < 100 {

									if messageParts[2] != MODE_PERCH && messageParts[2] != MODE_FLAP {
										penseFeatherPluckMap.Set(messageParts[3], true)
									}
									switch {
									case strings.HasPrefix(messageParts[2], MODE_PERCH): // Perch
										penseFeatherCtlCodeMap.Set(messageParts[3], messageParts[2])
										msg = MODE_PERCH
									case strings.HasPrefix(messageParts[2], MODE_FLAP): // Flap
										if strings.HasPrefix(msg, MODE_GAZE) { // If had gaze, then flap...
											penseFeatherCtlCodeMap.Set(messageParts[3], messageParts[2])
										}
									case strings.HasPrefix(messageParts[2], MODE_GAZE): // Gaze
										if msg != MODE_GLIDE { // Gliding to perch...
											penseFeatherCtlCodeMap.Set(messageParts[3], messageParts[2])
										} else {
											penseFeatherCtlCodeMap.Set(messageParts[3], MODE_PERCH)
										}
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
								penseFeatherCodeMap.Set(messageParts[1], "")
							}
						}
					}
				}
			}
		failover:
			conn.Write([]byte(" "))
			defer conn.Close()
			return
		} else {
			if _, ok := clientCodeMap.Get(conn.RemoteAddr().String()); !ok {
				clientCodeMap.Set(conn.RemoteAddr().String(), [][]byte{})
			}

			if ccmap, ok := clientCodeMap.Get(conn.RemoteAddr().String()); ok {
				clientCodeMap.Set(conn.RemoteAddr().String(), append(ccmap, append([]byte{}, buf[:n]...)))
			}
		}
	}
}

func Feather(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, acceptRemote func(int, string) bool) {
	go func() {
		if pluckListener, err := kcp.ListenWithOptions(hostAddr+"1", nil, 0, 0); err == nil {
			for {
				pluckS, err := pluckListener.AcceptKCP()
				if err != nil {
					continue
				}
				if acceptRemote(FEATHER_COMMON, pluckS.RemoteAddr().String()) {
					go handlePluck(pluckS, acceptRemote)
				} else {
					pluckS.Close()
				}
			}
		}
	}()
	key := pbkdf2.Key([]byte(encryptPass), []byte(encryptSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	if listener, err := kcp.ListenWithOptions(hostAddr, block, 10, 3); err == nil {
		for {
			s, err := listener.AcceptKCP()
			if err != nil {
				continue
			}
			if acceptRemote(FEATHER_COMMON, s.RemoteAddr().String()) {
				go handleMessage(handshakeCode, s, acceptRemote)
			} else {
				s.Close()
			}
		}
	}
}

// Pluck is a blocking call
func PluckCtlEmit(hostAddr string, pense string, acceptRemote func(int, string) (bool, error)) (bool, error) {

	for {
		penseConn, penseErr := kcp.Dial(hostAddr + "1")

		if penseErr != nil {
			time.Sleep(time.Second)
			continue
		}

		defer penseConn.Close()
		_, penseWriteErr := penseConn.Write([]byte(MODE_PLUCK + ":" + pense))
		if penseWriteErr != nil {
			time.Sleep(time.Second)
			continue
		}

		responseBuf := make([]byte, 100)

		penseConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, penseResponseErr := penseConn.Read(responseBuf)
		if penseResponseErr != nil {
			time.Sleep(time.Second)
			continue
		}
		response := string(responseBuf[:n])
		if response == MODE_PLUCK {
			return true, nil
		}

		if acceptRemote == nil {
			return false, nil
		} else {
			if breakImmediate, accErr := acceptRemote(FEATHER_CTL, penseConn.RemoteAddr().String()); breakImmediate {
				// Break, but don't exit encapsulating calling function.
				return false, accErr
			} else {
				// No break immediate, however only return if error is returned...
				if accErr != nil {
					return true, accErr
				}
			}
		}
	}
}

func FeatherCtlEmit(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, modeCtlPack string, pense string, bypass bool, acceptRemote func(int, string) (bool, error)) (string, error) {
	if !bypass && modeCtlPack == MODE_FLAP {
		if breakImmediate, accErr := PluckCtlEmit(hostAddr, pense, acceptRemote); breakImmediate && accErr != nil {
			return "", accErr
		}
	}
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

	responseBuf := make([]byte, 100)

	penseConn.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))
	n, penseResponseErr := penseConn.Read(responseBuf)

	return string(responseBuf[:n]), penseResponseErr
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

	responseBuf := make([]byte, 100)
	n, penseResponseErr := penseConn.Read(responseBuf)

	return responseBuf[:n], penseResponseErr
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

	if _, penseCodeOk := tap.PenseCode(penseCode); penseCodeOk {
		if pense, penseOk := penseMemoryMap[penseRequest.PenseIndex]; penseOk {
			return &PenseReply{Pense: pense}, nil
		} else {
			return &PenseReply{Pense: "Pense undefined"}, nil
		}
	} else {
		// Might be a feather
		if _, penseCodeOk := penseFeatherCodeMap.Get(penseCode); penseCodeOk {
			penseFeatherCodeMap.Remove(penseCode)
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
	go tap.Tap(brimPath, "f19431f322ea015ef871d267cc75e58b73d16617f9ff47ed7e0f0c1dbfb276b5", "", false)
	TapServer("127.0.0.1:1534")

}
