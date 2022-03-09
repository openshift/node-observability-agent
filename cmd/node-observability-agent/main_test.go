package main

import (
	"os"
	"testing"
)

type TestCase struct {
	name          string
	tokenFile     string
	nodeIP        string
	storageFolder string
	crioSocket    string
	expectPanic   bool
}

func TestReadTokenFile(t *testing.T) {
	invalidTokenFile := "/tmp/noToken"
	token := "abc"
	validTokenFile := "/tmp/aToken"
	err := os.WriteFile(validTokenFile, []byte(token), 0644)
	if err != nil {
		t.Error(err)
	}
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
			name:          "file not found, error",
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
	validTokenFile := "/tmp/aToken"
	err := os.WriteFile(validTokenFile, []byte("abc"), 0644)
	if err != nil {
		t.Error(err)
	}
	invalidTokenFile := "/tmp/noToken"
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

	validSocket := "/tmp/aSocket"
	_, err = os.Create(validSocket)
	if err != nil {
		t.Error(err)
	}
	err = os.Chmod(validSocket, 0755)
	if err != nil {
		t.Error(err)
	}
	invalidSocket := "/tmp/noSocket"
	unWriteableSocket := "/tmp/unWriteableSocket"
	_, err = os.Create(unWriteableSocket)
	if err != nil {
		t.Error(err)
	}
	err = os.Chmod(unWriteableSocket, 0444)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if os.Remove(validSocket) != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if os.Remove(unWriteableSocket) != nil {
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

	testCases := []TestCase{
		{
			name:          "All params are OK, no errors",
			tokenFile:     validTokenFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			crioSocket:    validSocket,
			expectPanic:   false,
		},
		{
			name:          "Token file doesnt exist, error",
			tokenFile:     invalidTokenFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			crioSocket:    validSocket,
			expectPanic:   true,
		},
		{
			name:          "Token file is not readeable, error",
			tokenFile:     unReadableTokenFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			crioSocket:    validSocket,
			expectPanic:   true,
		},
		// {
		// 	name: "Token file is empty, error",
		// },
		{
			name:          "nodeIP is an invalid IP, error",
			tokenFile:     validTokenFile,
			nodeIP:        " 1000.40.210.253",
			storageFolder: validStorageFolder,
			crioSocket:    validSocket,
			expectPanic:   true,
		},
		{
			name:          "storageFolder doesnt exist, error",
			tokenFile:     validTokenFile,
			nodeIP:        "127.0.0.1",
			storageFolder: invalidStorageFolder,
			crioSocket:    validSocket,
			expectPanic:   true,
		},
		{
			name:          "storageFolder is not writable, error",
			tokenFile:     validTokenFile,
			nodeIP:        "127.0.0.1",
			storageFolder: unWriteableStorageFolder,
			crioSocket:    validSocket,
			expectPanic:   true,
		},
		{
			name:          "crio socket file doesnt exist, error",
			tokenFile:     validTokenFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			crioSocket:    invalidSocket,
			expectPanic:   true,
		},
		{
			name:          "crio socket file is not writable, error",
			tokenFile:     validTokenFile,
			nodeIP:        "127.0.0.1",
			storageFolder: validStorageFolder,
			crioSocket:    unWriteableSocket,
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
	checkParameters(tc.tokenFile, tc.nodeIP, tc.storageFolder, tc.crioSocket)
}
