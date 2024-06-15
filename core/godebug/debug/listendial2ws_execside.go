//go:build !js && editorDebugExecSide

////go:build !js // DEBUG

package debug

import (
	"context"
	"fmt"
	"net"
)

// NOTE: not supporting having a dependency on golang.org/x/net/websocket for godebug compilation
func acceptWebsocket(conn net.Conn) (net.Conn, error) {
	return nil, fmt.Errorf("not supported: !js && execside")
}
func dialWebsocket(ctx context.Context, addr Addr, conn net.Conn) (Conn, error) {
	return nil, fmt.Errorf("not supported: !js && execside")
}
