// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2020 The FluxCD contributors. All rights reserved.
// This package provides an in-memory known hosts database
// derived from the golang.org/x/crypto/ssh/knownhosts
// package.
// It has been slightly modified and adapted to work with
// in-memory host keys not related to any known_hosts files
// on disk, and the database can be initialized with just a
// known_hosts byte blob.
// https://pkg.go.dev/golang.org/x/crypto/ssh/knownhosts

package knownhosts

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// See the sshd manpage
// (http://man.openbsd.org/sshd#SSH_KNOWN_HOSTS_FILE_FORMAT) for
// background.

type addr struct{ host, port string }

func (a *addr) String() string {
	h := a.host
	if strings.Contains(h, ":") {
		h = "[" + h + "]"
	}
	return h + ":" + a.port
}

type matcher interface {
	match(addr) bool
}

type hostPattern struct {
	negate bool
	addr   addr
}

func (p *hostPattern) String() string {
	n := ""
	if p.negate {
		n = "!"
	}

	return n + p.addr.String()
}

type hostPatterns []hostPattern

func (ps hostPatterns) match(a addr) bool {
	matched := false
	for _, p := range ps {
		if !p.match(a) {
			continue
		}
		if p.negate {
			return false
		}
		matched = true
	}
	return matched
}

// See
// https://android.googlesource.com/platform/external/openssh/+/ab28f5495c85297e7a597c1ba62e996416da7c7e/addrmatch.c
// The matching of * has no regard for separators, unlike filesystem globs
func wildcardMatch(pat []byte, str []byte) bool {
	for {
		if len(pat) == 0 {
			return len(str) == 0
		}
		if len(str) == 0 {
			return false
		}

		if pat[0] == '*' {
			if len(pat) == 1 {
				return true
			}

			for j := range str {
				if wildcardMatch(pat[1:], str[j:]) {
					return true
				}
			}
			return false
		}

		if pat[0] == '?' || pat[0] == str[0] {
			pat = pat[1:]
			str = str[1:]
		} else {
			return false
		}
	}
}

func (p *hostPattern) match(a addr) bool {
	return wildcardMatch([]byte(p.addr.host), []byte(a.host)) && p.addr.port == a.port
}

type inMemoryHostKeyDB struct {
	hostKeys []hostKey
	revoked  map[string]*ssh.PublicKey
}

func newInMemoryHostKeyDB() *inMemoryHostKeyDB {
	db := &inMemoryHostKeyDB{
		revoked: make(map[string]*ssh.PublicKey),
	}

	return db
}

func keyEq(a, b ssh.PublicKey) bool {
	return bytes.Equal(a.Marshal(), b.Marshal())
}

type hostKey struct {
	matcher matcher
	cert    bool
	key     ssh.PublicKey
}

func (l *hostKey) match(a addr) bool {
	return l.matcher.match(a)
}

// IsAuthorityForHost can be used as a callback in ssh.CertChecker
func (db *inMemoryHostKeyDB) IsHostAuthority(remote ssh.PublicKey, address string) bool {
	h, p, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	a := addr{host: h, port: p}

	for _, l := range db.hostKeys {
		if l.cert && keyEq(l.key, remote) && l.match(a) {
			return true
		}
	}
	return false
}

// IsRevoked can be used as a callback in ssh.CertChecker
func (db *inMemoryHostKeyDB) IsRevoked(key *ssh.Certificate) bool {
	_, ok := db.revoked[string(key.Marshal())]
	return ok
}

const markerCert = "@cert-authority"
const markerRevoked = "@revoked"

func nextWord(line []byte) (string, []byte) {
	i := bytes.IndexAny(line, "\t ")
	if i == -1 {
		return string(line), nil
	}

	return string(line[:i]), bytes.TrimSpace(line[i:])
}

func parseLine(line []byte) (marker, host string, key ssh.PublicKey, err error) {
	if w, next := nextWord(line); w == markerCert || w == markerRevoked {
		marker = w
		line = next
	}

	host, line = nextWord(line)
	if len(line) == 0 {
		return "", "", nil, errors.New("knownhosts: missing host pattern")
	}

	// ignore the keytype as it's in the key blob anyway.
	_, line = nextWord(line)
	if len(line) == 0 {
		return "", "", nil, errors.New("knownhosts: missing key type pattern")
	}

	keyBlob, _ := nextWord(line)

	keyBytes, err := base64.StdEncoding.DecodeString(keyBlob)
	if err != nil {
		return "", "", nil, err
	}
	key, err = ssh.ParsePublicKey(keyBytes)
	if err != nil {
		return "", "", nil, err
	}

	return marker, host, key, nil
}

func (db *inMemoryHostKeyDB) parseLine(line []byte) error {
	marker, pattern, key, err := parseLine(line)
	if err != nil {
		return err
	}

	if marker == markerRevoked {
		db.revoked[string(key.Marshal())] = &key
		return nil
	}

	entry := hostKey{
		key:  key,
		cert: marker == markerCert,
	}

	if pattern[0] == '|' {
		entry.matcher, err = newHashedHost(pattern)
	} else {
		entry.matcher, err = newHostnameMatcher(pattern)
	}

	if err != nil {
		return err
	}

	db.hostKeys = append(db.hostKeys, entry)
	return nil
}

func newHostnameMatcher(pattern string) (matcher, error) {
	var hps hostPatterns
	for _, p := range strings.Split(pattern, ",") {
		if len(p) == 0 {
			continue
		}

		var a addr
		var negate bool
		if p[0] == '!' {
			negate = true
			p = p[1:]
		}

		if len(p) == 0 {
			return nil, errors.New("knownhosts: negation without following hostname")
		}

		var err error
		if p[0] == '[' {
			a.host, a.port, err = net.SplitHostPort(p)
			if err != nil {
				return nil, err
			}
		} else {
			a.host, a.port, err = net.SplitHostPort(p)
			if err != nil {
				a.host = p
				a.port = "22"
			}
		}
		hps = append(hps, hostPattern{
			negate: negate,
			addr:   a,
		})
	}
	return hps, nil
}

// check checks a key against the host database. This should not be
// used for verifying certificates.
func (db *inMemoryHostKeyDB) check(address string, remote net.Addr, remoteKey ssh.PublicKey) error {
	if revoked := db.revoked[string(remoteKey.Marshal())]; revoked != nil {
		return &knownhosts.RevokedError{Revoked: knownhosts.KnownKey{Key: *revoked}}
	}

	host, port, err := net.SplitHostPort(remote.String())
	if err != nil {
		return fmt.Errorf("knownhosts: SplitHostPort(%s): %v", remote, err)
	}

	hostToCheck := addr{host, port}
	if address != "" {
		// Give preference to the hostname if available.
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("knownhosts: SplitHostPort(%s): %v", address, err)
		}

		hostToCheck = addr{host, port}
	}

	return db.checkAddr(hostToCheck, remoteKey)
}

// checkAddr checks if we can find the given public key for the
// given address.  If we only find an entry for the IP address,
// or only the hostname, then this still succeeds.
func (db *inMemoryHostKeyDB) checkAddr(a addr, remoteKey ssh.PublicKey) error {
	// TODO(hanwen): are these the right semantics? What if there
	// is just a key for the IP address, but not for the
	// hostname?

	// Algorithm => key.
	knownKeys := map[string]ssh.PublicKey{}
	for _, l := range db.hostKeys {
		if l.match(a) {
			typ := l.key.Type()
			if _, ok := knownKeys[typ]; !ok {
				knownKeys[typ] = l.key
			}
		}
	}

	keyErr := &knownhosts.KeyError{}
	for _, v := range knownKeys {
		keyErr.Want = append(keyErr.Want, knownhosts.KnownKey{Key: v})
	}

	// Unknown remote host.
	if len(knownKeys) == 0 {
		return keyErr
	}

	// If the remote host starts using a different, unknown key type, we
	// also interpret that as a mismatch.
	if known, ok := knownKeys[remoteKey.Type()]; !ok || !keyEq(known, remoteKey) {
		return keyErr
	}

	return nil
}

// The Read function parses file contents.
func (db *inMemoryHostKeyDB) Read(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		if err := db.parseLine(line); err != nil {
			return fmt.Errorf("knownhosts: %v", err)
		}
	}
	return scanner.Err()
}

// New creates a host key callback from the given OpenSSH host key
// file bytes. The returned callback is for use in
// ssh.ClientConfig.HostKeyCallback. By preference, the key check
// operates on the hostname if available, i.e. if a server changes its
// IP address, the host key check will still succeed, even though a
// record of the new IP address is not available.
func New(b []byte) (ssh.HostKeyCallback, error) {
	db := newInMemoryHostKeyDB()
	r := bytes.NewReader(b)
	if err := db.Read(r); err != nil {
		return nil, err
	}

	var certChecker ssh.CertChecker
	certChecker.IsHostAuthority = db.IsHostAuthority
	certChecker.IsRevoked = db.IsRevoked
	certChecker.HostKeyFallback = db.check

	return certChecker.CheckHostKey, nil
}

func decodeHash(encoded string) (hashType string, salt, hash []byte, err error) {
	if len(encoded) == 0 || encoded[0] != '|' {
		err = errors.New("knownhosts: hashed host must start with '|'")
		return
	}
	components := strings.Split(encoded, "|")
	if len(components) != 4 {
		err = fmt.Errorf("knownhosts: got %d components, want 3", len(components))
		return
	}

	hashType = components[1]
	if salt, err = base64.StdEncoding.DecodeString(components[2]); err != nil {
		return
	}
	if hash, err = base64.StdEncoding.DecodeString(components[3]); err != nil {
		return
	}
	return
}

func encodeHash(typ string, salt []byte, hash []byte) string {
	return strings.Join([]string{"",
		typ,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hash),
	}, "|")
}

// See https://android.googlesource.com/platform/external/openssh/+/ab28f5495c85297e7a597c1ba62e996416da7c7e/hostfile.c#120
func hashHost(hostname string, salt []byte) []byte {
	mac := hmac.New(sha1.New, salt)
	mac.Write([]byte(hostname))
	return mac.Sum(nil)
}

type hashedHost struct {
	salt []byte
	hash []byte
}

const sha1HashType = "1"

func newHashedHost(encoded string) (*hashedHost, error) {
	typ, salt, hash, err := decodeHash(encoded)
	if err != nil {
		return nil, err
	}

	// The type field seems for future algorithm agility, but it's
	// actually hardcoded in openssh currently, see
	// https://android.googlesource.com/platform/external/openssh/+/ab28f5495c85297e7a597c1ba62e996416da7c7e/hostfile.c#120
	if typ != sha1HashType {
		return nil, fmt.Errorf("knownhosts: got hash type %s, must be '1'", typ)
	}

	return &hashedHost{salt: salt, hash: hash}, nil
}

func (h *hashedHost) match(a addr) bool {
	return bytes.Equal(hashHost(knownhosts.Normalize(a.String()), h.salt), h.hash)
}
