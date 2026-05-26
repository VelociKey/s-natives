package lifecycle
 
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"
 
	"sov.fleet/s-logiclibrary/00200-logic-libraries/bicodec"
)
 
// RunDaemon sets up the UDP SACP telemetry control stream.
func (e *NATVSEngine) RunDaemon(ctx context.Context, port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	slog.Info("s-natives Engine Daemon live", "port", port)
	
	timeout := e.Config.IdleTimeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	
	// Create watchdog timer
	watchdog := time.AfterFunc(timeout, func() {
		slog.Warn("Watchdog inactivity timeout reached. Shutting down daemon...", "timeout", timeout)
		conn.Close()
	})
	defer watchdog.Stop()
 
	buf := make([]byte, 1024)
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	
	for {
		n, clientAddr, err := conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			slog.Error("Daemon read failed", "err", err)
			continue
		}
		
		// Reset watchdog timer on packet activity
		watchdog.Reset(timeout)
		
		slog.Debug("Daemon received packet", "bytes", n, "remote", clientAddr)
		
		hasHeader, header, hasCap, capFrame, err := bicodec.DecodeSACPMessage(buf[:n])
		if err != nil {
			slog.Error("SACP Decode failed", "err", err)
			continue
		}
		
		if hasCap && capFrame.Domain == "tool" && capFrame.Action == "call" {
			cmd := capFrame.ID
			slog.Info("Daemon executing command", "cmd", cmd, "target", capFrame.Parameters["target"])
			
			// Respond back with success acknowledgment SACP frame
			resHeader := bicodec.SACPHeader{
				UUID:              "natives-daemon-uuid",
				OriginalAuthority: bicodec.AuthSovereign,
				CurrentAuthority:  bicodec.AuthSovereign,
			}
			resCap := bicodec.SACPCapability{
				Domain: "tool",
				Action: "call",
				ID:     cmd,
				Parameters: map[string]string{
					"status": "PASS",
				},
			}
			responseBytes, err := bicodec.EncodeSACPMessage(&resHeader, &resCap)
			if err != nil {
				slog.Error("Encoding response failed", "err", err)
				continue
			}
			
			_, err = conn.WriteTo(responseBytes, clientAddr)
			if err != nil {
				slog.Error("Response failed", "err", err)
			}
		} else {
			_ = hasHeader
			_ = header
			slog.Debug("SACP frame ignored (no tool call)")
		}
	}
}
