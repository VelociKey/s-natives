package lifecycle

const NetbenchGo = `package netbench

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"sov.fleet/s-logiclibrary/00200-logic-libraries/bicodec"
	"sov.fleet/s-logiclibrary/00200-logic-libraries/gatekeeper"
)

// MsgFormat defines the protocol format.
type MsgFormat int

const (
	weBNF MsgFormat = iota
	SACPUnsecured
	SACPSecured
)

// TestMessage contains the fields we send and verify.
type TestMessage struct {
	UUID               string
	OriginalAuthority  string
	CurrentAuthority   string
	AttestationType    string
	AttestationPayload string
	Domain             string
	Action             string
	ID                 string
	Parameters         map[string]string
	Version            string
}

// DefaultMessage returns a standard test message populated with security metadata.
func DefaultMessage() TestMessage {
	return TestMessage{
		UUID:               "msg-uuid-123456",
		OriginalAuthority:  bicodec.AuthSovereign,
		CurrentAuthority:   bicodec.AuthSovereign,
		AttestationType:    bicodec.AttestationVeracity,
		AttestationPayload: "veracity-seal-signature-payload-data",
		Domain:             "tool",
		Action:             "call",
		ID:                 "read_file",
		Parameters:         map[string]string{"path": "C:\\aCogSpaceSeed\\file.txt"},
		Version:            "1.0.0",
	}
}

// SerializeText serializes the message to weBNF text format.
func SerializeText(m TestMessage) []byte {
	var params []string
	for k, v := range m.Parameters {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}
	pStr := strings.Join(params, ";") + ";"
	s := fmt.Sprintf("LPSV_ENVELOPE {\n  uuid = \"%s\" .\n  original_authority = \"%s\" .\n  current_authority = \"%s\" .\n  attestation_type = \"%s\" .\n  attestation_payload = \"%s\" .\n  domain = \"%s\" .\n  action = \"%s\" .\n  id = \"%s\" .\n  parameters = \"%s\" .\n  version = \"%s\" .\n}", m.UUID, m.OriginalAuthority, m.CurrentAuthority, m.AttestationType, m.AttestationPayload, m.Domain, m.Action, m.ID, pStr, m.Version)
	return []byte(s)
}

// DeserializeText parses weBNF text format.
func DeserializeText(data []byte) (TestMessage, error) {
	s := string(data)
	if !strings.HasPrefix(s, "LPSV_ENVELOPE {") || !strings.HasSuffix(s, "}") {
		return TestMessage{}, errors.New("invalid weBNF header/footer")
	}
	m := TestMessage{Parameters: make(map[string]string)}
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "LPSV_ENVELOPE {" || line == "}" {
			continue
		}
		eq := strings.Index(line, " = ")
		if eq < 0 {
			continue
		}
		key := line[:eq]
		val := line[eq+3:]
		// Strip quotes and ending dot
		if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, " .") {
			val = val[1 : len(val)-3]
		} else if strings.HasSuffix(val, " .") {
			val = val[:len(val)-2]
		}
		switch key {
		case "uuid":
			m.UUID = val
		case "original_authority":
			m.OriginalAuthority = val
		case "current_authority":
			m.CurrentAuthority = val
		case "attestation_type":
			m.AttestationType = val
		case "attestation_payload":
			m.AttestationPayload = val
		case "domain":
			m.Domain = val
		case "action":
			m.Action = val
		case "id":
			m.ID = val
		case "version":
			m.Version = val
		case "parameters":
			parts := strings.Split(val, ";")
			for _, part := range parts {
				if part == "" {
					continue
				}
				peq := strings.Index(part, "=")
				if peq >= 0 {
					m.Parameters[part[:peq]] = part[peq+1:]
				}
			}
		}
	}
	return m, nil
}

// SerializeSACPUnsecured encodes message to raw SACP binary bytes.
func SerializeSACPUnsecured(m TestMessage) ([]byte, error) {
	h := bicodec.SACPHeader{
		UUID:               m.UUID,
		OriginalAuthority:  m.OriginalAuthority,
		CurrentAuthority:   m.CurrentAuthority,
		AttestationType:    m.AttestationType,
		AttestationPayload: m.AttestationPayload,
	}
	c := bicodec.SACPCapability{
		Domain:     m.Domain,
		Action:     m.Action,
		ID:         m.ID,
		Parameters: m.Parameters,
		Version:    m.Version,
	}
	return bicodec.EncodeSACPMessage(&h, &c)
}

// DeserializeSACPUnsecured parses SACP binary bytes.
func DeserializeSACPUnsecured(data []byte) (TestMessage, error) {
	hasH, h, hasC, c, err := bicodec.DecodeSACPMessage(data)
	if err != nil {
		return TestMessage{}, err
	}
	if !hasH || !hasC {
		return TestMessage{}, errors.New("missing SACP header or capability frame")
	}
	return TestMessage{
		UUID:               h.UUID,
		OriginalAuthority:  h.OriginalAuthority,
		CurrentAuthority:   h.CurrentAuthority,
		AttestationType:    h.AttestationType,
		AttestationPayload: h.AttestationPayload,
		Domain:             c.Domain,
		Action:             c.Action,
		ID:                 c.ID,
		Parameters:         c.Parameters,
		Version:            c.Version,
	}, nil
}

var nonceCounter uint64

// SerializeSACPSecured serializes to SACP binary enveloped/signed format.
func SerializeSACPSecured(m TestMessage, auditor *gatekeeper.CryptosealAuditor, secret []byte) ([]byte, error) {
	if auditor == nil {
		return nil, errors.New("pointer safety error: gatekeeper auditor is nil")
	}
	rawBytes, err := SerializeSACPUnsecured(m)
	if err != nil {
		return nil, err
	}
	payloadStr := hex.EncodeToString(rawBytes)
	now := time.Now()
	ctr := atomic.AddUint64(&nonceCounter, 1)
	nonce := fmt.Sprintf("nonce-%d-%d", now.UnixNano(), ctr)
	env, err := auditor.SignEnvelope(payloadStr, nonce, now, secret)
	if err != nil {
		return nil, err
	}
	sigLen := uint32(len(env.Signature))
	nonceLen := uint32(len(env.Nonce))
	totalHeaderSize := 4 + sigLen + 4 + nonceLen + 8
	buf := make([]byte, totalHeaderSize+uint32(len(rawBytes)))

	binary.BigEndian.PutUint32(buf[0:4], sigLen)
	copy(buf[4:4+sigLen], env.Signature)
	pos := 4 + sigLen

	binary.BigEndian.PutUint32(buf[pos:pos+4], nonceLen)
	copy(buf[pos+4:pos+4+nonceLen], env.Nonce)
	pos += 4 + nonceLen

	binary.BigEndian.PutUint64(buf[pos:pos+8], uint64(now.Unix()))
	pos += 8

	copy(buf[pos:], rawBytes)
	return buf, nil
}

// DeserializeSACPSecured parses and cryptographically validates the signed SACP envelope.
func DeserializeSACPSecured(data []byte, gk *gatekeeper.Gatekeeper, secret []byte, maxAge time.Duration) (TestMessage, error) {
	if gk == nil {
		return TestMessage{}, errors.New("pointer safety error: gatekeeper is nil")
	}
	if len(data) < 16 {
		return TestMessage{}, errors.New("payload too small")
	}
	sigLen := binary.BigEndian.Uint32(data[0:4])
	if uint32(len(data)) < 4+sigLen {
		return TestMessage{}, errors.New("invalid signature length boundary")
	}
	sig := string(data[4 : 4+sigLen])
	pos := 4 + sigLen

	nonceLen := binary.BigEndian.Uint32(data[pos : pos+4])
	if uint32(len(data)) < pos+4+nonceLen {
		return TestMessage{}, errors.New("invalid nonce length boundary")
	}
	nonce := string(data[pos+4 : pos+4+nonceLen])
	pos += 4 + nonceLen

	tsUnix := int64(binary.BigEndian.Uint64(data[pos : pos+8]))
	pos += 8

	rawSACP := data[pos:]

	payloadStr := hex.EncodeToString(rawSACP)
	env := gatekeeper.SignedEnvelope{
		Payload:   payloadStr,
		Nonce:     nonce,
		Timestamp: time.Unix(tsUnix, 0),
		Signature: sig,
	}

	err := gk.InterrogateEnvelope(env, secret, maxAge)
	if err != nil {
		return TestMessage{}, fmt.Errorf("envelope verification failed: %w", err)
	}

	return DeserializeSACPUnsecured(rawSACP)
}
`
