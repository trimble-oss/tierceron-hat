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

	"github.com/mrjrieke/hat/cap/tap"

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
	MODE_FLAP  = "p"
	MODE_GLIDE = "g"
	MODE_GAZE  = "z"
)

var penseMemoryMap map[string]string = map[string]string{}

var penseFeatherCodeMap = cmap.New[string]()
var penseFeatherMemoryMap map[string]string = map[string]string{}

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
						messageBytes = cremote[0]
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

								if len(messageParts[3]) < 20 && len(messageParts[2]) < 100 {
									switch {
									case strings.HasPrefix(messageParts[2], MODE_PERCH): // Perch
										penseFeatherCtlCodeMap.Set(messageParts[3], messageParts[2])
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
			conn.Write([]byte(" "))
			defer conn.Close()
			return
		} else {
			if ccmap, ok := clientCodeMap.Get(conn.RemoteAddr().String()); ok {
				clientCodeMap.Set(conn.RemoteAddr().String(), append(ccmap, append([]byte{}, buf[:n]...)))
			}
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
	go tap.Tap(brimPath, "f19431f322ea015ef871d267cc75e58b73d16617f9ff47ed7e0f0c1dbfb276b5")
	TapServer("127.0.0.1:1534")

}
