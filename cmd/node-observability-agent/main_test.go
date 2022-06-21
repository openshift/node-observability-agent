package main

import (
	"os"
	"testing"
)

type TestCase struct {
	name          string
	tokenFile     string
	caCertFile    string
	nodeIP        string
	storageFolder string
	expectPanic   bool
}

func TestMakeCACertPool(t *testing.T) {
	// #nosec G101 this is just a test file, cotaining random text
	invalidCACertFile := "/tmp/notACert"
	err := os.WriteFile(invalidCACertFile, []byte("not a cert"), 0600)
	if err != nil {
		t.Error(err)
	}
	// #nosec G101 this is just a test file, cotaining random text
	validCACertFile := "../../test_resources/kubelet-serving-ca.crt"

	// #nosec G101 this is just an empty test file
	emptyCAFile := "/tmp/emptyCA"
	_, err = os.Create(emptyCAFile)
	if err != nil {
		t.Error(err)
	}

	defer func() {
		if os.Remove(invalidCACertFile) != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if os.Remove(emptyCAFile) != nil {
			t.Error(err)
		}
	}()

	testCases := []struct {
		name          string
		caCertFile    string
		expectedError bool
	}{
		{
			name:          "CACertFile readeable, no errors",
			caCertFile:    validCACertFile,
			expectedError: false,
		},
		{
			name:          "CACertFile file not found, error",
			caCertFile:    "/tmp/CertNotExist.crt",
			expectedError: true,
		},
		{
			name:          "CACertFile empty, error",
			caCertFile:    emptyCAFile,
			expectedError: true,
		},
		{
			name:          "CACertFile invalid content, error",
			caCertFile:    invalidCACertFile,
			expectedError: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cacert, err := makeCACertPool(tc.caCertFile)
			if tc.expectedError {
				if err == nil {
					t.Error("Expected error but didnt get any")

				}
			} else {
				if err != nil {
					t.Errorf("Did not expect error but got %s", err.Error())
				}
				if len(cacert.Subjects()) == 0 {
					t.Error("cacert pool should contain at least one subject")
				}
			}
		})
	}
}
func TestReadTokenFile(t *testing.T) {
	// #nosec G101 this is just a test file, cotaining random text
	invalidTokenFile := "/tmp/noToken"
	token := "abc"
	// #nosec G101 this is just a test file, cotaining random text
	validTokenFile := "/tmp/aToken"
	err := os.WriteFile(validTokenFile, []byte(token), 0600)
	if err != nil {
		t.Error(err)
	}
	// #nosec G101 this is just an empty test file
	emptyTokenFile := "/tmp/emptyToken"
	_, err = os.Create(emptyTokenFile)
	if err != nil {
		t.Error(err)
	}

	defer func() {
		if os.Remove(validTokenFile) != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if os.Remove(emptyTokenFile) != nil {
			t.Error(err)
		}
	}()

	testCases := []struct {
		name          string
		tokenFile     string
		expectedToken string
		expectedError bool
	}{
		{
			name:          "token readeable, no errors",
			tokenFile:     validTokenFile,
			expectedToken: "abc",
			expectedError: false,
		},
		{
			name:          "token file not found, error",
			tokenFile:     invalidTokenFile,
			expectedToken: "",
			expectedError: true,
		},
		{
			name:          "token empty, error",
			tokenFile:     emptyTokenFile,
			expectedToken: "",
			expectedError: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := readTokenFile(tc.tokenFile)
			if tc.expectedToken != token {
				t.Errorf("expected returned token %s, but was %s", tc.expectedToken, token)
			}
			if tc.expectedError {
				if err == nil {
					t.Error("Expected error but didnt get any")

				}
			} else {
				if err != nil {
					t.Errorf("Did not expect error but got %s", err.Error())
				}
			}
		})
	}
}
func TestCheckParameters(t *testing.T) {
	// #nosec G101 this is just an empty test file
	validTokenFile := "/tmp/aToken"
	err := os.WriteFile(validTokenFile, []byte("abc"), 0600)
	if err != nil {
		t.Error(err)
	}
	// #nosec G101 this is just an empty test file
	invalidTokenFile := "/tmp/noToken"
	// #nosec G101 this is just an empty test file
	unReadableTokenFile := "/tmp/noReadToken"
	_, err = os.Create(unReadableTokenFile)
	if err != nil {
		t.Error(err)
	}
	err = os.Chmod(unReadableTokenFile, 0311)
	if err != nil {
		t.Error(err)
	}

	defer func() {
		if os.Remove(validTokenFile) != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if os.Remove(unReadableTokenFile) != nil {
			t.Error(err)
		}
	}()

	validStorageFolder := "/tmp/aFolder"
	err = os.Mkdir(validStorageFolder, 0755)
	if err != nil {
		t.Error(err)
	}
	invalidStorageFolder := "/tmp/noFolder"
	unWriteableStorageFolder := "/tmp/unWriteableFolder"
	err = os.Mkdir(unWriteableStorageFolder, 0555)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(validStorageFolder)
	defer os.Remove(unWriteableStorageFolder)

	// #nosec G101 this is just a test file, cotaining random text
	invalidCACertFile := "/tmp/noCert"
	cert := "A Fake Cert Content"
	// #nosec G101 this is just a test file, cotaining random text
	validCACertFile := "/tmp/aCA"
	err = os.WriteFile(validCACertFile, []byte(cert), 0400)
	if err != nil {
		t.Error(err)
	}
	// #nosec G101 this is just an empty test file
	unReadableCAFile := "/tmp/unReadableCA"
	_, err = os.Create(unReadableCAFile)
	if err != nil {
		t.Error(err)
	}
	err = os.Chmod(unReadableCAFile, 0100)
	if err != nil {
		t.Error(err)
	}

	defer func() {
		if os.Remove(validCACertFile) != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if os.Remove(unReadableCAFile) != nil {
			t.Error(err)
		}
	}()

	testCases := []TestCase{
		{
			name:          "All params are OK, no errors",
			tokenFile:     validTokenFile,
			caCertFile:    validCACertFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			expectPanic:   false,
		},
		{
			name:          "Token file doesnt exist, error",
			tokenFile:     invalidTokenFile,
			caCertFile:    validCACertFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			expectPanic:   true,
		},
		{
			name:          "Token file is not readeable, error",
			tokenFile:     unReadableTokenFile,
			caCertFile:    validCACertFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			expectPanic:   true,
		},
		{
			name:          "nodeIP is an invalid IP, error",
			tokenFile:     validTokenFile,
			caCertFile:    validCACertFile,
			nodeIP:        " 1000.40.210.253",
			storageFolder: validStorageFolder,
			expectPanic:   true,
		},
		{
			name:          "storageFolder doesnt exist, error",
			tokenFile:     validTokenFile,
			caCertFile:    validCACertFile,
			nodeIP:        "127.0.0.1",
			storageFolder: invalidStorageFolder,
			expectPanic:   true,
		},
		{
			name:          "storageFolder is not writable, error",
			tokenFile:     validTokenFile,
			caCertFile:    validCACertFile,
			nodeIP:        "127.0.0.1",
			storageFolder: unWriteableStorageFolder,
			expectPanic:   true,
		},
		{
			name:          "CACert file doesnt exist, error",
			tokenFile:     validTokenFile,
			caCertFile:    invalidCACertFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			expectPanic:   true,
		},
		{
			name:          "CACert file is not readeable, error",
			tokenFile:     validTokenFile,
			caCertFile:    unReadableCAFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			expectPanic:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checkPanic(t, tc)
		})
	}
}

func checkPanic(t *testing.T, tc TestCase) {
	defer func() {
		if tc.expectPanic {
			if recover() == nil {
				t.Errorf("The code did not panic when it was expected to")
			}
		} else {
			if recover() != nil {
				t.Errorf("The code panicked when it wasnt expected to")
			}
		}
	}()
	checkParameters(tc.tokenFile, tc.nodeIP, tc.storageFolder, tc.caCertFile)
}
