package lifecycle

const RsaGenPattern = `	// 1. Generate ephemeral private keys
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client key: %w", err)
	}`

const EcdsaGenReplacement = `	// 1. Generate ephemeral private keys (Optimized to ECDSA P-256 for zero-overhead loopback)
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client key: %w", err)
	}`

const GatekeeperStructPattern = `// Gatekeeper is the Zero-Trust Enforcer.
type Gatekeeper struct {
	mu         sync.Mutex
	seenNonces map[string]time.Time
	auditor    *CryptosealAuditor
}`

const GatekeeperStructReplacement = `type nonceEntry struct {
	nonce     string
	timestamp time.Time
}

// Gatekeeper is the Zero-Trust Enforcer.
type Gatekeeper struct {
	mu         sync.Mutex
	seenNonces map[string]time.Time
	nonceQueue []nonceEntry // Optimized O(1) queue-based pruning
	auditor    *CryptosealAuditor
}`

const NewGatekeeperPattern = `func NewGatekeeper() *Gatekeeper {
	return &Gatekeeper{
		seenNonces: make(map[string]time.Time),
		auditor:    NewCryptosealAuditor(),
	}
}`

const NewGatekeeperReplacement = `func NewGatekeeper() *Gatekeeper {
	return &Gatekeeper{
		seenNonces: make(map[string]time.Time),
		nonceQueue: make([]nonceEntry, 0),
		auditor:    NewCryptosealAuditor(),
	}
}`

const PrunePattern = `	// Prune expired nonces to control memory footprints
	for nonce, ts := range g.seenNonces {
		if now.Sub(ts) > maxAge {
			delete(g.seenNonces, nonce)
		}
	}

	if _, exists := g.seenNonces[env.Nonce]; exists {
		return errors.New("REJECTED: Replay attack detected. Nonce already processed")
	}

	// Register nonce
	g.seenNonces[env.Nonce] = env.Timestamp`

const PruneReplacement = `	// Prune expired nonces to control memory footprints (O(1) queue-based pruning)
	for len(g.nonceQueue) > 0 && now.Sub(g.nonceQueue[0].timestamp) > maxAge {
		expired := g.nonceQueue[0].nonce
		delete(g.seenNonces, expired)
		g.nonceQueue = g.nonceQueue[1:]
	}

	if _, exists := g.seenNonces[env.Nonce]; exists {
		return errors.New("REJECTED: Replay attack detected. Nonce already processed")
	}

	// Register nonce
	g.seenNonces[env.Nonce] = env.Timestamp
	g.nonceQueue = append(g.nonceQueue, nonceEntry{nonce: env.Nonce, timestamp: env.Timestamp})`
