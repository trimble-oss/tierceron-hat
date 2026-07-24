package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/trimble-oss/tierceron-hat/cap"
	"github.com/trimble-oss/tierceron-hat/localcert"
)

func loadLocalFeatherTLSConfig(serverName string) (*cap.FeatherTLSConfig, error) {
	return localcert.LoadFeatherTLSConfig(
		[]string{"./servicecert.crt", "./local_config/servicecert.crt", "../servicecert.crt", "../local_config/servicecert.crt", "./serv_cert.pem", "../serv_cert.pem"},
		[]string{"./servicekey.key", "./local_config/servicekey.key", "../servicekey.key", "../local_config/servicekey.key", "./serv_key.pem", "../serv_key.pem"},
		[]string{"./serviceclientcert.pem", "./local_config/serviceclientcert.pem", "../serviceclientcert.pem", "../local_config/serviceclientcert.pem", "./servicecert.crt", "./local_config/servicecert.crt", "../servicecert.crt", "../local_config/servicecert.crt", "./serv_cert.pem", "../serv_cert.pem"},
		serverName,
	)
}

// The next crowning
func main() {
	featherServerName := flag.String("fsn", "", "TLS server name covered by the local feather certificate")
	flag.Parse()

	fmt.Println("Starting tiara")
	tlsConfig, err := loadLocalFeatherTLSConfig(*featherServerName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	go cap.FeatherWithTLS("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", tlsConfig, func(int, string) bool { return true })

	keyvar := new(string)
	*keyvar = "therefore I am."
	cap.TapFeather("I think", keyvar)

	keyvar2 := new(string)
	*keyvar2 = "The main thing is to use it well."
	cap.TapFeather("It is not enough to have a good mind.", keyvar2)

	keyvar3 := new(string)
	*keyvar3 = "me this."
	cap.TapMemorize("Ponder", keyvar3)

	keyvar4 := new(string)
	*keyvar4 = "a feather."
	cap.TapFeather("Ponder", keyvar4)

	cap.TapServer("127.0.0.1:1534")
}
