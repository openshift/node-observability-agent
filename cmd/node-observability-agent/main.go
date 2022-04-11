package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/node-observability-agent/pkg/server"
)

var (
	version       = "unknown"
	app           = "node-observability-agent"
	node          = os.Getenv("NODE_IP")
	port          = flag.Int("port", 9000, "server port to listen on (default: 9000)")
	storageFolder = flag.String("storage", "/tmp/pprofs/", "folder to which the pprof files are saved")
	tokenFile     = flag.String("tokenFile", "", "file containing token to be used for kubelet profiling http request")
	caCertFile    = flag.String("caCertFile", "/var/run/secrets/kubernetes.io/serviceaccount/kubelet-serving-ca.crt", "file containing CAChain for verifying the TLS certificate on kubelet profiling http request")
	crioSocket    = flag.String("crioUnixSocket", "/var/run/crio/crio.sock", "file referring to the unix socket to be used for CRIO profiling")
	logLevel      = flag.String("loglevel", "info", "log level")
	versionFlag   = flag.Bool("v", false, "print version")
	appVersion    = fmt.Sprintf("%s %s", app, version)
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(appVersion)
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
	log.Infof("Starting %s at log level %s", appVersion, *logLevel)

	checkParameters(*tokenFile, node, *storageFolder, *crioSocket, *caCertFile)

	/* #nosec G304 tokenFile is a parameter of the agent’s go program.
	*  Upon creation of the NodeObservability CR, the operator creates a SA for the agent, sets its RBAC,
	* and provides the tokenFile parameter in the daemonset manifest: The value provided is the default file
	* kubernetes mounts on the node containing the SA JWT)
	* The agent only takes the token file from that
	 */
	token, err := readTokenFile(*tokenFile)
	if err != nil {
		panic("Unable to read token file, or token is empty :" + err.Error())
	}

	caCerts, err := readCACertsFile(*caCertFile)
	if err != nil {
		panic("Unable to read caCerts file :" + err.Error())
	}

	server.Start(server.Config{
		Port:           *port,
		Token:          token,
		CACerts:        caCerts,
		StorageFolder:  *storageFolder,
		CrioUnixSocket: *crioSocket,
		NodeIP:         node,
	})
}

func checkParameters(tokenFile string, nodeIP string, storageFolder string, crioUnixSocket string, caCertFile string) {
	//check on configs that are passed along before starting up the server
	//1. token is readable
	_, err := readTokenFile(tokenFile)
	if err != nil {
		panic("Unable to read token file")
	}
	//2. nodeIP is found
	if nodeIP == "" || net.ParseIP(nodeIP) == nil {
		panic("Environment variable NODE_IP not found, or doesnt contain a valid IP address")
	}
	//3. StorageFolder is accessible in readwrite
	if syscall.Access(storageFolder, syscall.O_RDWR) != nil {
		panic("Unable to access the folder specified for saving the profiling data - no write permission :" + storageFolder)
	}
	//4. CRIO socket is accessible in readwrite
	if syscall.Access(crioUnixSocket, syscall.O_RDWR) != nil {
		panic("Unable to access the the crio socket - no write permission :" + crioUnixSocket)
	}
	//5. CACerts file is readable
	const R_OK uint32 = 4
	if syscall.Access(caCertFile, R_OK) != nil {
		panic("Unable to read the caCerts file :" + caCertFile)
	}
}

func readTokenFile(tokenFile string) (string, error) {
	content, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	if len(content) <= 0 {
		return "", fmt.Errorf("%s was empty", tokenFile)
	}
	return string(content), nil
}

func readCACertsFile(caCertFile string) ([]byte, error) {
	content, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return content, err
	}
	if len(content) <= 0 {
		err = fmt.Errorf("%s was empty", caCertFile)
	}
	return content, err
}
