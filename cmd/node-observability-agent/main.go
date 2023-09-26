package main

import (
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/node-observability-agent/pkg/server"
	ver "github.com/openshift/node-observability-agent/pkg/version"
)

var (
	nodeIP               = os.Getenv("NODE_IP")
	port                 = flag.Int("port", 9000, "server port to listen on (default: 9000)")
	unixSocket           = flag.String("unixSocket", "/var/run/nobagent.sock", "unix socket to listen on (default: /var/run/nobagent.sock)")
	preferUnixSocket     = flag.Bool("preferUnixSocket", false, "listen on the unix socket instead of localhost:port")
	storageFolder        = flag.String("storage", "/tmp/pprofs/", "folder to which the pprof files are saved")
	tokenFile            = flag.String("tokenFile", "", "file containing token to be used for kubelet profiling http request")
	caCertFile           = flag.String("caCertFile", "/var/run/secrets/kubernetes.io/serviceaccount/kubelet-serving-ca.crt", "file containing CAChain for verifying the TLS certificate on kubelet profiling http request")
	crioUnixSocket       = flag.String("crioUnixSocket", "/var/run/crio/crio.sock", "file referring to the unix socket to be used for CRIO profiling")
	crioPreferUnixSocket = flag.Bool("crioPreferUnixSocket", true, "use unix socket to communicate to CRIO")
	logLevel             = flag.String("loglevel", "info", "log level")
	versionFlag          = flag.Bool("v", false, "print version")
	mode                 = flag.String("mode", "profiling", "flag (profiling or scripting) to set mode (crio,kubelet) profiling or metrics script execution")
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(ver.MakeVersionString())
		os.Exit(0)
	}

	// Gosec G304 (CWE-22) - Mitigated
	// This is a parameter passed via a command line. The agent only takes the token file from this parameter
	// and cannot be changed as it is not exposed via an environmental variable , configmap or secret
	lvl, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("Log level %s not recognized, using info", *logLevel)
		*logLevel = "info"
		lvl = log.InfoLevel
	}
	log.SetLevel(lvl)
	log.Infof("Starting %s at log level %s", ver.MakeVersionString(), *logLevel)

	checkParameters(*mode, nodeIP, *storageFolder, *crioUnixSocket, *crioPreferUnixSocket, *caCertFile)

	var token string
	var caCerts *x509.CertPool
	if *mode == "profiling" {
		/* #nosec G304 tokenFile is a parameter of the agent’s go program.
		*  Upon creation of the NodeObservability CR, the operator creates a SA for the agent, sets its RBAC,
		* and provides the tokenFile parameter in the daemonset manifest: The value provided is the default file
		* kubernetes mounts on the node containing the SA JWT)
		* The agent only takes the token file from that
		 */
		token, err = readTokenFile(*tokenFile)
		if err != nil {
			panic("Unable to read token file, or token is empty :" + err.Error())
		}

		caCerts, err = makeCACertPool(*caCertFile)
		if err != nil {
			panic("Unable to read caCerts file :" + err.Error())
		}
	}

	if err := server.Start(server.Config{
		Port:                 *port,
		UnixSocket:           *unixSocket,
		PreferUnixSocket:     *preferUnixSocket,
		Token:                token,
		CACerts:              caCerts,
		StorageFolder:        *storageFolder,
		CrioUnixSocket:       *crioUnixSocket,
		CrioPreferUnixSocket: *crioPreferUnixSocket,
		NodeIP:               nodeIP,
		Mode:                 *mode,
	}); err != nil {
		log.Errorf("Error from server: %s", err.Error())
	}
	log.Info("Stopped")
}

func checkParameters(mode, nodeIP, storageFolder, crioUnixSocket string, crioPreferUnixSocket bool, caCertFile string) {
	//check on configs that are passed along before starting up the server
	// nodeIP is found
	if nodeIP == "" || net.ParseIP(nodeIP) == nil {
		panic("Environment variable NODE_IP not found, or doesn't contain a valid IP address")
	}
	// StorageFolder is accessible in readwrite
	if err := syscall.Access(storageFolder, syscall.O_RDWR); err != nil {
		panic(fmt.Sprintf("Unable to access the storage folder for saving the profiling data %q: %v", storageFolder, err))
	}

	if mode == "profiling" {
		// CRIO socket is accessible in readwrite
		if crioPreferUnixSocket {
			if err := syscall.Access(crioUnixSocket, syscall.O_RDWR); err != nil {
				panic(fmt.Sprintf("Unable to access the crio socket %q: %v", crioUnixSocket, err))
			}
		}
		// CACerts file is readable
		const R_OK uint32 = 4
		if syscall.Access(caCertFile, R_OK) != nil {
			panic("Unable to read the caCerts file :" + caCertFile)
		}

	} else if mode == "scripting" {
		if os.Getenv("EXECUTE_SCRIPT") == "" {
			panic("Ensure the EXECUTE_SCRIPT envar is set (name of script to execute)")
		}
	}
}

func readTokenFile(tokenFile string) (string, error) {
	content, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	if len(content) <= 0 {
		return "", fmt.Errorf("%s was empty", tokenFile)
	}
	return string(content), nil
}

func makeCACertPool(caCertFile string) (*x509.CertPool, error) {
	content, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, err
	}
	if len(content) <= 0 {
		return nil, fmt.Errorf("%s was empty", caCertFile)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(content) {
		return nil, fmt.Errorf("Unable to add certificates into caCertPool:\n%v", err)

	}
	return caCertPool, nil
}
