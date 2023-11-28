package cap

import (
	"bytes"
	context "context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"os"
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

var (
	MODE_PERCH byte = 'c'
	MOID_VOID  byte = 'v'
	MODE_PLUCK byte = 'k'
	MODE_FLAP  byte = 'p'
	MODE_GLIDE byte = 'g'
	MODE_GAZE  byte = 'z'

	PROTOCOL_DELIM byte = ':'
)

var (
	CTL_COMPLETE       string = "CTLCOMPLETE"
	CTL_COMPLETE_BYTES []byte = []byte(CTL_COMPLETE)
	PROTOCOL_HDR       string = "featherctl"
	PROTOCOL_HDR_BYTES []byte = []byte(PROTOCOL_HDR)
	MODE_FLAP_BYTES    []byte = []byte{MODE_FLAP}
	MODE_GLIDE_BYTES   []byte = []byte{MODE_GLIDE}
)

const (
	RUN_STARTED = 1 << iota // RUN_STARTED
	RUNNING     = 1 << iota // RUNNING 2
	RESETTING   = 1 << iota // RESETTING 4
)

type FeatherContext struct {
	EncryptPass                    *string
	EncryptSalt                    *string
	LocalHostAddr                  *string
	HostAddr                       *string
	HandshakeCode                  *string
	SessionIdentifier              *string
	AcceptRemoteFunc               func(*FeatherContext, int, string) (bool, error)
	InterruptHandlerFunc           func(*FeatherContext) error
	InterruptChan                  chan os.Signal
	RunState                       int64 // whether to restart a run
	TwoHundredMilliInterruptTicker *time.Ticker
	MultiSecondInterruptTicker     *time.Ticker
	FifteenSecondInterruptTicker   *time.Ticker
	ThirtySecondInterruptTicker    *time.Ticker
	Log                            log.Logger
}

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

func hasMode(msg []byte, mode byte) bool {
	for _, b := range msg {
		if b == '\x00' {
			continue
		} else if b == mode {
			return true
		} else {
			return false
		}

	}
	return false
}

func handlePluck(conn *kcp.UDPSession, acceptRemote func(int, string) bool) {
	buf := make([]byte, 50)
	for {
		conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			conn.Close()
			return
		}
		message := buf[:n]

		if hasMode(message, MODE_PLUCK) {
			message = bytes.TrimLeft(message, "\x00")
			if len(message) > 2 {
				if _, ok := penseFeatherPluckMap.Pop(string(message[2:])); ok {
					conn.Write([]byte{MODE_PLUCK})
					conn.Close()
					return
				} else {
					conn.Write([]byte{MOID_VOID})
					conn.Close()
					return
				}
			}
		} else {
			conn.Close()
			return
		}
	}
}

func bytesSplit(data []byte, separator byte) [][]byte {
	var parts [][]byte

	for start := 0; start < len(data); {
		end := start

		for end < len(data) && data[end] != separator {
			end++
		}

		part := data[start:end]
		parts = append(parts, part)
		start = end + 1
	}

	return parts
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
				}
				if err == nil {
					clientCodeMap.Set(conn.RemoteAddr().String(), [][]byte{})
					if acceptRemote(FEATHER_SECRET, conn.RemoteAddr().String()) {
						message := string(messageBytes)
						messageParts := strings.Split(message, string(PROTOCOL_DELIM))
						if messageParts[0] == handshakeCode {
							if len(messageParts[1]) == 64 {
								penseFeatherCodeMap.Set(messageParts[1], "")
							}
						}
					}
				}
			}
			conn.Write([]byte{' '})
			defer conn.Close()
			return
		} else {
			if _, ok := clientCodeMap.Get(conn.RemoteAddr().String()); !ok {
				clientCodeMap.Set(conn.RemoteAddr().String(), [][]byte{})
			}

			if bytes.HasPrefix(buf[:n], PROTOCOL_HDR_BYTES) {
				if acceptRemote(FEATHER_CTL, conn.RemoteAddr().String()) {
					message := buf[:n]
					messageParts := bytesSplit(message, PROTOCOL_DELIM)
					if bytes.HasPrefix([]byte(handshakeCode), messageParts[1]) && len(messageParts) == 4 {
						// featherctl:handshakecode:f|p|g:activity
						var msg string = ""
						var ok bool
						activity := string(messageParts[3])
						ctl := string(messageParts[2])
						if msg, ok = penseFeatherCtlCodeMap.Get(activity); !ok {
							// Default is Perch
							msg = string(MODE_PERCH)
						}

						if len(messageParts[3]) < 20 && len(messageParts[2]) < 100 {

							if len(messageParts[2]) > 0 && messageParts[2][0] != MODE_PERCH && messageParts[2][0] != MODE_FLAP {
								penseFeatherPluckMap.Set(activity, true)
							}
							switch {
							case len(messageParts[2]) > 0 && messageParts[2][0] == MODE_PERCH: // Perch
								penseFeatherCtlCodeMap.Set(activity, ctl)
								msg = string(MODE_PERCH)
							case len(messageParts[2]) > 0 && messageParts[2][0] == MODE_FLAP: // Flap
								if msg[0] == MODE_GAZE { // If had gaze, then flap...
									penseFeatherCtlCodeMap.Set(activity, ctl)
								}
							case len(messageParts[2]) > 0 && messageParts[2][0] == MODE_GAZE: // Gaze
								if msg[0] != MODE_GLIDE { // Gliding to perch...
									penseFeatherCtlCodeMap.Set(activity, ctl)
								} else {
									penseFeatherCtlCodeMap.Set(activity, string(MODE_PERCH))
								}
							case len(messageParts[2]) > 0 && messageParts[2][0] == MODE_GLIDE: // Glide
								penseFeatherCtlCodeMap.Set(activity, ctl)
							}
						}
						conn.Write([]byte(msg))
						defer conn.Close()
						return
					}
				}
				conn.Write([]byte{' '})
				defer conn.Close()
				return
			} else {
				if ccmap, ok := clientCodeMap.Get(conn.RemoteAddr().String()); ok {
					clientCodeMap.Set(conn.RemoteAddr().String(), append(ccmap, append([]byte{}, buf[:n]...)))
				}
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
func PluckCtlEmit(featherCtx *FeatherContext, pense []byte) (bool, error) {

	pluckPacket := make([]byte, 2+len(pense)+1)
	pluckPacket = append(pluckPacket, MODE_PLUCK)
	pluckPacket = append(pluckPacket, PROTOCOL_DELIM)
	pluckPacket = append(pluckPacket, pense...)

	hostAddr := *featherCtx.HostAddr + "1"
	responseBuf := make([]byte, 100)

	for {
		penseConn, penseErr := kcp.Dial(hostAddr)
		if penseErr != nil {
			time.Sleep(time.Second)
			continue
		}

		defer penseConn.Close()
		_, penseWriteErr := penseConn.Write(pluckPacket)
		if penseWriteErr != nil {
			time.Sleep(time.Second)
			continue
		}

		penseConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, penseResponseErr := penseConn.Read(responseBuf)
		if penseResponseErr != nil {
			time.Sleep(time.Second)
			continue
		}

		response := responseBuf[:n]
		if hasMode(response, MODE_PLUCK) {
			return true, nil
		}

		if featherCtx.AcceptRemoteFunc == nil {
			return false, nil
		} else {
			if breakImmediate, accErr := featherCtx.AcceptRemoteFunc(featherCtx, FEATHER_CTL, penseConn.RemoteAddr().String()); breakImmediate {
				// Break, but don't exit encapsulating calling function.
				if accErr != nil {
					return true, accErr
				} else {
					return false, accErr
				}
			} else {
				// No break immediate, however only return if error is returned...
				if accErr != nil {
					return true, accErr
				}
			}
		}
	}
}

func FeatherCtlEmitBinary(featherCtx *FeatherContext, modeCtlPack string, pense []byte, bypass bool) ([]byte, error) {
	if !bypass && modeCtlPack[0] == MODE_FLAP {
		if breakImmediate, accErr := PluckCtlEmit(featherCtx, pense); breakImmediate && accErr != nil {
			return nil, accErr
		}
	}

	key := pbkdf2.Key([]byte(*featherCtx.EncryptPass), []byte(*featherCtx.EncryptSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	penseConn, penseErr := kcp.DialWithOptions(*featherCtx.HostAddr, block, 10, 3)
	if penseErr != nil {
		return nil, penseErr
	}
	defer penseConn.Close()
	packet := []byte(PROTOCOL_HDR)
	packet = append(packet, PROTOCOL_DELIM)
	packet = append(packet, []byte(*featherCtx.HandshakeCode)...)
	packet = append(packet, PROTOCOL_DELIM)
	packet = append(packet, []byte(modeCtlPack)...)
	packet = append(packet, PROTOCOL_DELIM)
	packet = append(packet, pense...)
	_, penseWriteErr := penseConn.Write(packet)
	if penseWriteErr != nil {
		return nil, penseWriteErr
	}

	responseBuf := make([]byte, 100)

	penseConn.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))
	n, penseResponseErr := penseConn.Read(responseBuf)

	return responseBuf[:n], penseResponseErr

}

func FeatherCtlEmit(featherCtx *FeatherContext, modeCtlPack string, pense string, bypass bool) (string, error) {
	response, err := FeatherCtlEmitBinary(featherCtx, modeCtlPack, []byte(pense), bypass)
	if response != nil {
		return string(response), err
	} else {
		return "", err
	}
}

func FeatherWriter(featherCtx *FeatherContext, pense string) ([]byte, error) {
	penseSplits, err := shamir.Split([]byte(*featherCtx.HandshakeCode+string(PROTOCOL_DELIM)+pense), 12, 7)
	if err != nil {
		return nil, err
	}
	key := pbkdf2.Key([]byte(*featherCtx.EncryptPass), []byte(*featherCtx.EncryptSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	penseConn, penseErr := kcp.DialWithOptions(*featherCtx.HostAddr, block, 10, 3)
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
