package storage_test

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/url"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/andrewhowdencom/x40.link/storage"
	storer "github.com/andrewhowdencom/x40.link/storage/firestore"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// pids is a registry of processes that need to be shutdown. Used primarily for the firebase emulator.
var pids = struct {
	mu sync.Mutex
	m  map[string]*exec.Cmd
}{
	m: make(map[string]*exec.Cmd),
}

// Factories to generate valid storage engines
var externalSinkFactories = map[string]func(string) storage.Storer{
	"firestore": func(s string) storage.Storer {
		// Start the firebase emulator
		cmd := exec.Command(
			"gcloud",
			"emulators",
			"firestore",
			"start",
			"--host-port=localhost:8500",
		)

		// Set process group ID
		// See https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// Save the STDOUT & STDERR
		cmd.Stdout = &bytes.Buffer{}
		cmd.Stderr = &bytes.Buffer{}

		if err := cmd.Start(); err != nil {
			panic(err)
		}

		// Wait for firestore to come up.
		ctx, cxl := context.WithTimeout(context.Background(), time.Second*60)
		defer cxl()

		ticker := time.NewTicker(time.Second * 1)

		i := 0
	Wait:
		for {
			i++

			select {
			case <-ctx.Done():
				log.Println("command output")
				log.Println(cmd.Stdout.(*bytes.Buffer).String())
				log.Println(cmd.Stderr.(*bytes.Buffer).String())

				panic("waited for firebase to come up, but it did not")
			case <-ticker.C:
				conn, err := net.DialTimeout("tcp", "localhost:8500", time.Millisecond*500)
				if err != nil {
					log.Printf("tried to connect; failed: %s. attempt %d", err.Error(), i)
				} else {
					// Error ignored as this is a test, and it will be shutdown anyway.
					_ = conn.Close()
					log.Println("connection succeeded")
					break Wait
				}
			}
		}

		// It takes a second for firestore to come up.
		time.Sleep(time.Second * 5)

		pids.mu.Lock()
		defer pids.mu.Unlock()

		pids.m[s] = cmd

		conn, err := grpc.Dial("localhost:8500", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			panic(err)
		}

		client, err := firestore.NewClient(context.Background(), "andrewhowdencom", option.WithGRPCConn(conn))
		if err != nil {
			panic(err)
		}

		// Bootstrap the client
		return &storer.Firestore{
			Client: client,
		}
	},
}

// Factories to tear down valid storage engines
var externalSinkTeardown = map[string]func(string){
	"firestore": func(s string) {
		if err := syscall.Kill(-pids.m[s].Process.Pid, syscall.SIGINT); err != nil {
			panic(err)
		}
	},
}

// race is designed to stress the storage by using it concurrently, such that the go race detector can

// TestComplianceAll tests that the storages actually store and retrieve valid records in the (simplest) expected ways.
func TestComplianceExternalAll(t *testing.T) {
	for n, f := range externalSinkFactories {
		f := f
		n := n

		t.Run(n, func(t *testing.T) {
			str := f("compliance")
			defer externalSinkTeardown[n]("compliance")

			// Query for a record that doesn't exit, to ensure the data store will not panic.
			_, err := str.Get(&url.URL{Host: "x40"})

			assert.ErrorIs(t, err, storage.ErrNotFound)

			// Insert and query a record.
			assert.Nil(t, str.Put(&url.URL{Host: "x40"}, &url.URL{Host: "andrewhowden.com"}))

			res, err := str.Get(&url.URL{
				Host: "x40",
			})

			assert.Nil(t, err)
			assert.Equal(t, &url.URL{
				Host: "andrewhowden.com",
			}, res)
		})
	}
}
