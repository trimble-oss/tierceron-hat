package cap

import (
	"bytes"
	context "context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap/tap"

	"github.com/lafriks/go-shamir"
	cmap "github.com/orcaman/concurrent-map/v2"
	quic "github.com/quic-go/quic-go"
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
	featherQUICALPN       = "feather-quic"
	featherQUICServerName = "feather.local"
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
	Env                            *string
	TLSConfig                      *FeatherTLSConfig
	controlConnMu                  sync.Mutex
	controlConn                    *quic.Conn
	dataConnMu                     sync.Mutex
	dataConn                       *quic.Conn
	AcceptRemoteFunc               func(*FeatherContext, int, string) (bool, error)
	InterruptHandlerFunc           func(*FeatherContext) error
	InterruptChan                  chan os.Signal
	RunState                       int64 // whether to restart a run
	TwoHundredMilliInterruptTicker *time.Ticker
	MultiSecondInterruptTicker     *time.Ticker
	FifteenSecondInterruptTicker   *time.Ticker
	ThirtySecondInterruptTicker    *time.Ticker
	Log                            *log.Logger
}

type FeatherTLSConfig struct {
	AllowSelfSigned bool
	ListenerCertPEM *[]byte
	ListenerKeyPEM  *[]byte
	RootCertPEM     *[]byte
	ServerName      *string
}

func NewFeatherSelfSignedTLSConfig() *FeatherTLSConfig {
	serverName := featherQUICServerName
	return &FeatherTLSConfig{
		AllowSelfSigned: true,
		ServerName:      &serverName,
	}
}

func NewFeatherPEMTLSConfig(listenerCertPEM, listenerKeyPEM, rootCertPEM *[]byte) *FeatherTLSConfig {
	return &FeatherTLSConfig{
		ListenerCertPEM: listenerCertPEM,
		ListenerKeyPEM:  listenerKeyPEM,
		RootCertPEM:     rootCertPEM,
	}
}

var penseMemoryMap map[string]*string = map[string]*string{}

var (
	penseFeatherCodeMap                      = cmap.New[string]()
	penseFeatherMemoryMap map[string]*string = map[string]*string{}
)

var (
	penseFeatherPluckMap   = cmap.New[bool]()
	penseFeatherCtlCodeMap = cmap.New[string]()
)

type featherTLSCacheKey struct {
	encryptPass      string
	encryptSalt      string
	allowSelfSigned  bool
	listenerCertHash string
	listenerKeyHash  string
	rootCertHash     string
	serverName       string
}

type featherTLSCacheValue struct {
	server *tls.Config
	client *tls.Config
}

var featherTLSCache sync.Map

// featherDoneChan signals shutdown to Feather listeners
var (
	featherDoneChan   = make(chan struct{})
	featherDoneClosed = false
)

// FeatherStop closes the done channel to signal Feather goroutines to exit
func FeatherStop() {
	if !featherDoneClosed {
		close(featherDoneChan)
		featherDoneClosed = true
	}
}

// CodeSaltGuardFn is expected to return a hex.EncodeToString encoded salt
type CodeSaltGuardFunc func() string

var CodeSaltGuardFn CodeSaltGuardFunc = nil

func TapInitCodeSaltGuard(csgFn CodeSaltGuardFunc) {
	CodeSaltGuardFn = csgFn
}

type featherQUICStream struct {
	conn   *quic.Conn
	stream *quic.Stream
}

func (fqs *featherQUICStream) Read(p []byte) (int, error) {
	return fqs.stream.Read(p)
}

func (fqs *featherQUICStream) Write(p []byte) (int, error) {
	return fqs.stream.Write(p)
}

func (fqs *featherQUICStream) Close() error {
	return fqs.stream.Close()
}

func (fqs *featherQUICStream) LocalAddr() net.Addr {
	return fqs.conn.LocalAddr()
}

func (fqs *featherQUICStream) RemoteAddr() net.Addr {
	return fqs.conn.RemoteAddr()
}

func (fqs *featherQUICStream) SetDeadline(t time.Time) error {
	return fqs.stream.SetDeadline(t)
}

func (fqs *featherQUICStream) SetReadDeadline(t time.Time) error {
	return fqs.stream.SetReadDeadline(t)
}

func (fqs *featherQUICStream) SetWriteDeadline(t time.Time) error {
	return fqs.stream.SetWriteDeadline(t)
}

func buildQUICCertificate(encryptPass, encryptSalt string, serverName string) (tls.Certificate, *x509.Certificate, error) {
	if len(serverName) == 0 {
		serverName = featherQUICServerName
	}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	serialHash := sha256.Sum256([]byte(encryptPass + ":" + encryptSalt + ":feather-cert"))
	serialNumber := new(big.Int).SetBytes(serialHash[:])
	if serialNumber.Sign() == 0 {
		serialNumber = big.NewInt(1)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: serverName,
		},
		DNSNames:              []string{serverName},
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		NotBefore:             time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2044, time.January, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	leaf, err := x509.ParseCertificate(certDER)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	tlsCertificate := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
		Leaf:        leaf,
	}

	return tlsCertificate, leaf, nil
}

func hashTLSBytes(value *[]byte) string {
	if value == nil || len(*value) == 0 {
		return ""
	}
	sum := sha256.Sum256(*value)
	return hex.EncodeToString(sum[:])
}

func newFeatherTLSCacheKey(encryptPass, encryptSalt string, tlsConfig *FeatherTLSConfig) featherTLSCacheKey {
	cacheKey := featherTLSCacheKey{encryptPass: encryptPass, encryptSalt: encryptSalt}
	if tlsConfig != nil {
		cacheKey.allowSelfSigned = tlsConfig.AllowSelfSigned
		cacheKey.listenerCertHash = hashTLSBytes(tlsConfig.ListenerCertPEM)
		cacheKey.listenerKeyHash = hashTLSBytes(tlsConfig.ListenerKeyPEM)
		cacheKey.rootCertHash = hashTLSBytes(tlsConfig.RootCertPEM)
		if tlsConfig.ServerName != nil {
			cacheKey.serverName = *tlsConfig.ServerName
		}
	}
	return cacheKey
}

func loadQUICCertificate(certPEM, keyPEM []byte) (tls.Certificate, *x509.Certificate, error) {
	tlsCertificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	if len(tlsCertificate.Certificate) == 0 {
		return tls.Certificate{}, nil, errors.New("listener certificate PEM is empty")
	}
	leaf, err := x509.ParseCertificate(tlsCertificate.Certificate[0])
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	tlsCertificate.Leaf = leaf
	return tlsCertificate, leaf, nil
}

func getQUICTLSConfigs(encryptPass, encryptSalt string, tlsConfig *FeatherTLSConfig) (*tls.Config, *tls.Config, error) {
	cacheKey := newFeatherTLSCacheKey(encryptPass, encryptSalt, tlsConfig)
	if cached, ok := featherTLSCache.Load(cacheKey); ok {
		cachedValue := cached.(featherTLSCacheValue)
		return cachedValue.server.Clone(), cachedValue.client.Clone(), nil
	}

	allowSelfSigned := tlsConfig == nil || tlsConfig.AllowSelfSigned
	serverName := featherQUICServerName
	if tlsConfig != nil && tlsConfig.ServerName != nil && len(*tlsConfig.ServerName) > 0 {
		serverName = *tlsConfig.ServerName
	}
	var tlsCertificate tls.Certificate
	var leaf *x509.Certificate
	var err error
	if tlsConfig != nil && ((tlsConfig.ListenerCertPEM != nil && len(*tlsConfig.ListenerCertPEM) > 0) || (tlsConfig.ListenerKeyPEM != nil && len(*tlsConfig.ListenerKeyPEM) > 0)) {
		if tlsConfig.ListenerCertPEM == nil || len(*tlsConfig.ListenerCertPEM) == 0 || tlsConfig.ListenerKeyPEM == nil || len(*tlsConfig.ListenerKeyPEM) == 0 {
			return nil, nil, errors.New("listener certificate and key PEM must both be provided")
		}
		tlsCertificate, leaf, err = loadQUICCertificate(*tlsConfig.ListenerCertPEM, *tlsConfig.ListenerKeyPEM)
	} else {
		tlsCertificate, leaf, err = buildQUICCertificate(encryptPass, encryptSalt, serverName)
	}
	if err != nil {
		return nil, nil, err
	}
	expectedPeerCertificate := append([]byte{}, leaf.Raw...)
	rootCAs := x509.NewCertPool()
	switch {
	case tlsConfig != nil && tlsConfig.RootCertPEM != nil && len(*tlsConfig.RootCertPEM) > 0:
		if !rootCAs.AppendCertsFromPEM(*tlsConfig.RootCertPEM) {
			return nil, nil, errors.New("failed to parse root certificate PEM")
		}
	case allowSelfSigned && tlsConfig != nil && tlsConfig.ListenerCertPEM != nil && len(*tlsConfig.ListenerCertPEM) > 0:
		if !rootCAs.AppendCertsFromPEM(*tlsConfig.ListenerCertPEM) {
			return nil, nil, errors.New("failed to parse listener certificate PEM")
		}
	case allowSelfSigned:
		rootCAs.AddCert(leaf)
	default:
		return nil, nil, errors.New("root certificate PEM required when AllowSelfSigned is false")
	}

	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCertificate},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{featherQUICALPN},
	}
	clientTLSConfig := &tls.Config{
		RootCAs:    rootCAs,
		ServerName: serverName,
		MinVersion: tls.VersionTLS13,
		NextProtos: []string{featherQUICALPN},
	}
	if allowSelfSigned {
		clientTLSConfig.InsecureSkipVerify = true
		clientTLSConfig.VerifyConnection = func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) != 1 {
				return errors.New("unexpected QUIC peer certificate chain")
			}
			if !bytes.Equal(state.PeerCertificates[0].Raw, expectedPeerCertificate) {
				return errors.New("unexpected QUIC peer certificate")
			}
			return nil
		}
	}

	featherTLSCache.Store(cacheKey, featherTLSCacheValue{server: serverTLSConfig, client: clientTLSConfig})
	return serverTLSConfig.Clone(), clientTLSConfig.Clone(), nil
}

func newFeatherQUICConfig() *quic.Config {
	return &quic.Config{
		HandshakeIdleTimeout: 5 * time.Second,
		MaxIdleTimeout:       30 * time.Second,
		KeepAlivePeriod:      10 * time.Second,
		MaxIncomingStreams:   128,
	}
}

func dialQUICConn(addr string, clientTLSConfig *tls.Config) (*quic.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	quicConn, err := quic.DialAddr(ctx, addr, clientTLSConfig, newFeatherQUICConfig())
	if err != nil {
		return nil, err
	}
	return quicConn, nil
}

func openQUICStream(quicConn *quic.Conn) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := quicConn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}

	return &featherQUICStream{conn: quicConn, stream: stream}, nil
}

func dialQUIC(addr string, clientTLSConfig *tls.Config) (net.Conn, error) {
	quicConn, err := dialQUICConn(addr, clientTLSConfig)
	if err != nil {
		return nil, err
	}

	streamConn, err := openQUICStream(quicConn)
	if err != nil {
		quicConn.CloseWithError(0, "")
		return nil, err
	}

	return streamConn, nil
}

func (featherCtx *FeatherContext) getOrCreateQUICConn(control bool, clientTLSConfig *tls.Config) (*quic.Conn, error) {
	addr := *featherCtx.HostAddr
	var connMu *sync.Mutex
	var currentConn **quic.Conn

	if control {
		connMu = &featherCtx.controlConnMu
		currentConn = &featherCtx.controlConn
	} else {
		connMu = &featherCtx.dataConnMu
		currentConn = &featherCtx.dataConn
	}

	connMu.Lock()
	defer connMu.Unlock()

	if *currentConn != nil {
		if (*currentConn).Context().Err() == nil {
			return *currentConn, nil
		}
		*currentConn = nil
	}

	quicConn, err := dialQUICConn(addr, clientTLSConfig.Clone())
	if err != nil {
		return nil, err
	}
	*currentConn = quicConn
	return quicConn, nil
}

func (featherCtx *FeatherContext) openQUICClientStream(control bool, clientTLSConfig *tls.Config) (net.Conn, error) {
	quicConn, err := featherCtx.getOrCreateQUICConn(control, clientTLSConfig)
	if err != nil {
		return nil, err
	}

	streamConn, err := openQUICStream(quicConn)
	if err == nil {
		return streamConn, nil
	}

	var connMu *sync.Mutex
	var currentConn **quic.Conn
	if control {
		connMu = &featherCtx.controlConnMu
		currentConn = &featherCtx.controlConn
	} else {
		connMu = &featherCtx.dataConnMu
		currentConn = &featherCtx.dataConn
	}

	connMu.Lock()
	if *currentConn == quicConn {
		(*currentConn).CloseWithError(0, "")
		*currentConn = nil
	}
	connMu.Unlock()

	quicConn, err = featherCtx.getOrCreateQUICConn(control, clientTLSConfig)
	if err != nil {
		return nil, err
	}

	return openQUICStream(quicConn)
}

func acceptQUICStream(conn *quic.Conn) (net.Conn, error) {
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		conn.CloseWithError(0, "")
		return nil, err
	}
	return &featherQUICStream{conn: conn, stream: stream}, nil
}

func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, io.EOF) {
		return true
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return strings.Contains(err.Error(), "timeout")
}

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

var clientCodeMap = cmap.New[[]byte]()

func encodeFeatherPayload(payload []byte) []byte {
	framed := make([]byte, 2+len(payload))
	framed[0] = byte(len(payload) >> 8)
	framed[1] = byte(len(payload))
	copy(framed[2:], payload)
	return framed
}

func decodeFeatherPayloads(payload []byte) [][]byte {
	decoded := [][]byte{}
	for offset := 0; offset+2 <= len(payload); {
		blockLen := int(payload[offset])<<8 | int(payload[offset+1])
		offset += 2
		if offset+blockLen > len(payload) {
			break
		}
		decoded = append(decoded, append([]byte{}, payload[offset:offset+blockLen]...))
		offset += blockLen
	}
	return decoded
}

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

func handlePluck(conn net.Conn, acceptRemote func(int, string) bool) {
	buf := make([]byte, 50)
	for {
		select {
		case <-featherDoneChan:
			conn.Close()
			return
		default:
		}
		if acceptRemote(FEATHER_COMMON, conn.RemoteAddr().String()) {
			lastActivityTime := time.Now()
			for {
				select {
				case <-featherDoneChan:
					conn.Close()
					return
				default:
				}

				// Force timeout recovery if no activity for 30 seconds
				if time.Since(lastActivityTime) > 30*time.Second {
					conn.Close()
					return
				}

				time.Sleep(time.Second * 3)
				conn.SetDeadline(time.Now().Add(15 * time.Second))
				n, err := conn.Read(buf)
				if err != nil {
					conn.Close()
					return
				}

				// Update activity tracker on successful read
				lastActivityTime = time.Now()
				message := buf[:n]

				if hasMode(message, MODE_PLUCK) {
					message = bytes.TrimLeft(message, "\x00")
					if len(message) > 2 {
						if _, ok := penseFeatherPluckMap.Pop(string(message[2:])); ok {
							conn.Write([]byte{MODE_PLUCK})
							continue
						} else {
							conn.Write([]byte{MOID_VOID})
							continue
						}
					}
				} else {
					continue
				}
			}
		} else {
			conn.Close()
			break
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

func handleMessage(handshakeCode string, conn net.Conn, acceptRemote func(int, string) bool) {
	buf := make([]byte, 4096)
	lastActivityTime := time.Now()
	for {
		select {
		case <-featherDoneChan:
			conn.Close()
			return
		default:
		}

		// Force timeout recovery if no activity for 30 seconds
		if time.Since(lastActivityTime) > 30*time.Second {
			conn.Close()
			return
		}

		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(buf)
		if _, ok := clientCodeMap.Get(conn.RemoteAddr().String()); !ok {
			clientCodeMap.Set(conn.RemoteAddr().String(), []byte{})
		}

		if n == 0 || err != nil {
			// All done... hopefully.
			if _, ok := clientCodeMap.Get(conn.RemoteAddr().String()); ok {
				var messageBytes []byte
				var err error = nil
				if cremote, ok := clientCodeMap.Get(conn.RemoteAddr().String()); ok && len(cremote) > 0 {
					decodedPayloads := decodeFeatherPayloads(cremote)
					if len(decodedPayloads) > 1 {
						messageBytes, err = shamir.Combine(decodedPayloads...)
					}
				}
				if err == nil {
					clientCodeMap.Set(conn.RemoteAddr().String(), []byte{})
					if acceptRemote(FEATHER_SECRET, conn.RemoteAddr().String()) {
						message := string(messageBytes)
						messageParts := strings.Split(message, string(PROTOCOL_DELIM))
						if messageParts[0] == handshakeCode {
							featherCode := messageParts[1]
							if CodeSaltGuardFn != nil {
								codeSalt := CodeSaltGuardFn()
								if len(codeSalt) > 0 {
									if strings.HasSuffix(featherCode, codeSalt) {
										featherCode = strings.TrimSuffix(featherCode, codeSalt)
									} else {
										// Invalid
										featherCode = ""
									}
								}

							}
							if len(featherCode) == 64 {
								penseFeatherCodeMap.Set(featherCode, "")
							}
						}
					}
				}
			}
			conn.Write([]byte{' '})
			defer conn.Close()
			return
		} else {
			// Update activity on successful read
			lastActivityTime = time.Now()
			if _, ok := clientCodeMap.Get(conn.RemoteAddr().String()); !ok {
				clientCodeMap.Set(conn.RemoteAddr().String(), []byte{})
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
								if msg[0] == MODE_GLIDE && bytes.HasSuffix([]byte(msg), CTL_COMPLETE_BYTES) {
									msg = string(MODE_GAZE)
									penseFeatherCtlCodeMap.Set(activity, msg)
								} else if msg[0] == MODE_GAZE || msg[0] == MODE_FLAP { // Preserve payload-bearing flaps once the session is active.
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
					clientCodeMap.Set(conn.RemoteAddr().String(), append(ccmap, buf[:n]...))
				}
				defer conn.Close()
			}
		}
	}
}

func Feather(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, acceptRemote func(int, string) bool) {
	FeatherWithTLS(encryptPass, encryptSalt, hostAddr, handshakeCode, nil, acceptRemote)
	return
}

func FeatherWithTLS(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, tlsConfig *FeatherTLSConfig, acceptRemote func(int, string) bool) {
	serverTLSConfig, _, err := getQUICTLSConfigs(encryptPass, encryptSalt, tlsConfig)
	if err != nil {
		return
	}

	go func() {
		if pluckListener, err := quic.ListenAddr(hostAddr+"1", serverTLSConfig.Clone(), newFeatherQUICConfig()); err == nil {
			go func() {
				<-featherDoneChan
				pluckListener.Close()
			}()
			for {
				pluckConn, err := pluckListener.Accept(context.Background())
				if err != nil {
					if errors.Is(err, quic.ErrServerClosed) {
						return
					}
					time.Sleep(time.Second)
					continue
				}
				go func(conn *quic.Conn) {
					for {
						streamConn, err := acceptQUICStream(conn)
						if err != nil {
							return
						}
						go handlePluck(streamConn, acceptRemote)
					}
				}(pluckConn)
			}
		}
	}()
	if listener, err := quic.ListenAddr(hostAddr, serverTLSConfig.Clone(), newFeatherQUICConfig()); err == nil {
		go func() {
			<-featherDoneChan
			listener.Close()
		}()
		for {
			conn, err := listener.Accept(context.Background())
			if err != nil {
				if errors.Is(err, quic.ErrServerClosed) {
					return
				}
				continue
			}
			go func(quicConn *quic.Conn) {
				for {
					streamConn, err := acceptQUICStream(quicConn)
					if err != nil {
						return
					}
					if acceptRemote(FEATHER_COMMON, streamConn.RemoteAddr().String()) {
						go handleMessage(handshakeCode, streamConn, acceptRemote)
					} else {
						streamConn.Close()
					}
				}
			}(conn)
		}
	}
}

// Pluck is a blocking call
func PluckCtlEmit(featherCtx *FeatherContext, pense []byte) (bool, error) {
	pluckPacket := []byte{MODE_PLUCK, PROTOCOL_DELIM}
	pluckPacket = append(pluckPacket, pense...)
	hostAddr := *featherCtx.HostAddr + "1"
	responseBuf := make([]byte, 100)
	_, clientTLSConfig, tlsErr := getQUICTLSConfigs(valueOrEmpty(featherCtx.EncryptPass), valueOrEmpty(featherCtx.EncryptSalt), featherCtx.TLSConfig)
	if tlsErr != nil {
		return true, tlsErr
	}

	var penseConn net.Conn
	var penseErr error
	retries := 0

retryEstablish:
	penseConn, penseErr = dialQUIC(hostAddr, clientTLSConfig.Clone())
	if penseErr == nil {
	}
	if penseErr != nil {
		time.Sleep(time.Second)
		if retries < 10 && penseErr != io.EOF {
			retries = retries + 1
			if penseConn != nil {
				penseConn.Close()
			}
			goto retryEstablish
		} else {
			// break immediately
			return true, penseErr
		}
	}

	defer penseConn.Close()

	for {
		time.Sleep(3 * time.Second)
		penseConn.SetDeadline(time.Now().Add(5 * time.Second))
		_, penseWriteErr := penseConn.Write(pluckPacket)
		if penseWriteErr != nil {
			if isTimeoutErr(penseWriteErr) {
				if retries < 10 {
					time.Sleep(time.Second)
					retries = retries + 1
					penseConn.Close()
					goto retryEstablish
				} else {
					// break immediately
					return true, penseWriteErr
				}
			}
			continue
		}

		penseConn.SetDeadline(time.Now().Add(5 * time.Second))
		n, penseResponseErr := penseConn.Read(responseBuf)
		if penseResponseErr != nil {
			if isTimeoutErr(penseResponseErr) {
				if retries < 10 {
					time.Sleep(time.Second)
					retries = retries + 1
					penseConn.Close()
					goto retryEstablish
				} else {
					// break immediately
					penseConn.Close()
					return true, penseResponseErr
				}
			}
			continue
		}
		retries = 0

		response := responseBuf[:n]
		if hasMode(response, MODE_PLUCK) {
			return true, nil
		}

		if featherCtx.AcceptRemoteFunc == nil {
			return false, nil
		} else {
			if breakImmediate, accErr := featherCtx.AcceptRemoteFunc(featherCtx, FEATHER_CTL, penseConn.RemoteAddr().String()); breakImmediate {
				if accErr != nil {
					return true, accErr
				} else {
					// Break, but don't exit encapsulating calling function.
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
	_, clientTLSConfig, tlsErr := getQUICTLSConfigs(valueOrEmpty(featherCtx.EncryptPass), valueOrEmpty(featherCtx.EncryptSalt), featherCtx.TLSConfig)
	if tlsErr != nil {
		return nil, tlsErr
	}

	penseConn, penseErr := featherCtx.openQUICClientStream(true, clientTLSConfig)
	if penseErr != nil {
		return nil, penseErr
	}
	defer penseConn.Close()
	// Preallocate enough space for all the pieces
	protocolSize := len(PROTOCOL_HDR) + 1 + len(*featherCtx.HandshakeCode) + 1 + len(modeCtlPack) + 1 + len(pense)
	packet := make([]byte, 0, protocolSize)

	packet = append(packet, PROTOCOL_HDR...)
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
	if n > 0 && errors.Is(penseResponseErr, io.EOF) {
		penseResponseErr = nil
	}

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
	// Create the message that will be split
	message := *featherCtx.HandshakeCode + string(PROTOCOL_DELIM) + pense
	penseSplits, err := shamir.Split([]byte(message), 12, 7)
	if err != nil {
		return nil, err
	}
	_, clientTLSConfig, tlsErr := getQUICTLSConfigs(valueOrEmpty(featherCtx.EncryptPass), valueOrEmpty(featherCtx.EncryptSalt), featherCtx.TLSConfig)
	if tlsErr != nil {
		return nil, tlsErr
	}

	penseConn, penseErr := featherCtx.openQUICClientStream(false, clientTLSConfig)
	if penseErr != nil {
		return nil, penseErr
	}
	defer penseConn.Close()
	for _, penseBlock := range penseSplits {
		_, penseWriteErr := penseConn.Write(encodeFeatherPayload(penseBlock))
		if penseWriteErr != nil {
			return nil, penseWriteErr
		}
	}

	responseBuf := make([]byte, 100)
	penseConn.SetReadDeadline(time.Now().Add(12 * time.Second))
	n, penseResponseErr := penseConn.Read(responseBuf)
	if n > 0 && errors.Is(penseResponseErr, io.EOF) {
		penseResponseErr = nil
	}

	return responseBuf[:n], penseResponseErr
}

func TapFeather(penseIndex string, memory *string) {
	penseMemoryMap[penseIndex] = memory
	penseFeatherMemoryMap[penseIndex] = memory
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func TapMemorize(penseIndex string, memory *string) {
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
			return &PenseReply{Pense: *pense}, nil
		} else {
			return &PenseReply{Pense: "Pense undefined"}, nil
		}
	} else {
		// Might be a feather
		if _, penseCodeOk := penseFeatherCodeMap.Get(penseCode); penseCodeOk {
			penseFeatherCodeMap.Remove(penseCode)
			if pense, penseOk := penseFeatherMemoryMap[penseRequest.PenseIndex]; penseOk {
				return &PenseReply{Pense: *pense}, nil
			} else {
				return &PenseReply{Pense: "Pense undefined"}, nil
			}
		}
		return &PenseReply{Pense: "...."}, nil
	}
}
