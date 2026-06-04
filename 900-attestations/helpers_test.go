package natives_attestations

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func findWorkspaceRoot() string {
	if root := os.Getenv("TEST_WORKSPACE_ROOT"); root != "" {
		return root
	}
	cwd, err := os.Getwd()
	if err == nil {
		dir := cwd
		for {
			if _, err := os.Stat(filepath.Join(dir, ".gitroot")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "C:\\aCogSpaceSeed"
}

func tempDirInWorkspace(t *testing.T) string {
	root := findWorkspaceRoot()
	baseDir := filepath.Join(root, "00flow", "s-natives", "c0990-ephemeral-scratch", "test-temp")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Logf("Warning: MkdirAll failed for baseDir: %v", err)
	}
	name := "test-" + t.Name()
	dir := filepath.Join(baseDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create test temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Warning: failed to remove temp dir %s: %v", dir, err)
		}
	})
	return dir
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func getExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

type EchoBackend struct{}

func (b *EchoBackend) Initialize(ctx context.Context) error { return nil }
func (b *EchoBackend) Validate(ctx context.Context) error   { return nil }
func (b *EchoBackend) Dial(ctx context.Context) (net.Conn, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	go func() {
		defer l.Close()
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 1024)
		for {
			n, err := c.Read(buf)
			if err != nil {
				return
			}
			if _, err := c.Write([]byte("ACK: " + string(buf[:n]))); err != nil {
				return
			}
		}
	}()
	return net.Dial("tcp", l.Addr().String())
}
func (b *EchoBackend) Shutdown(ctx context.Context) error { return nil }

func getWorkspaceRoot() string {
	workspaceRoot := os.Getenv("TEST_WORKSPACE_ROOT")
	if workspaceRoot == "" {
		cwd, err := os.Getwd()
		if err == nil {
			dir := cwd
			for {
				if _, err := os.Stat(filepath.Join(dir, ".gitroot")); err == nil {
					return dir
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
		}
		workspaceRoot = "C:\\aCogSpaceSeed"
	}
	return workspaceRoot
}

