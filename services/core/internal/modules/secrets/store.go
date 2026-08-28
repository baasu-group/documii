// Package secrets owns encryption of connector credentials. Callers receive
// plaintext only at the adapter boundary; HTTP and reporting packages must
// never read encrypted_secrets directly.
package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("secret not found")

type Store struct {
	pool   *pgxpool.Pool
	cipher *Cipher
}

type Cipher struct {
	keys    map[int][]byte
	current int
}

func NewCipher(current int, keys map[int][]byte) (*Cipher, error) {
	if current < 1 {
		return nil, errors.New("secret key version required")
	}
	copyKeys := map[int][]byte{}
	for version, key := range keys {
		if len(key) != 32 {
			return nil, fmt.Errorf("secret key version %d must be 32 bytes", version)
		}
		copyKeys[version] = append([]byte(nil), key...)
	}
	if copyKeys[current] == nil {
		return nil, errors.New("current secret key missing")
	}
	return &Cipher{keys: copyKeys, current: current}, nil
}
func (c *Cipher) Encrypt(label string, value []byte) ([]byte, []byte, int, error) {
	if label == "" || len(value) == 0 {
		return nil, nil, 0, errors.New("invalid secret")
	}
	ciphertext, nonce, err := seal(c.keys[c.current], value, []byte(label))
	return ciphertext, nonce, c.current, err
}
func (c *Cipher) Decrypt(label string, ciphertext, nonce []byte, version int) ([]byte, error) {
	key := c.keys[version]
	if key == nil {
		return nil, fmt.Errorf("secret key version %d unavailable", version)
	}
	return open(key, ciphertext, nonce, []byte(label))
}

func NewStore(pool *pgxpool.Pool, current int, keys map[int][]byte) (*Store, error) {
	if pool == nil {
		return nil, errors.New("secret database required")
	}
	cipher, err := NewCipher(current, keys)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool, cipher: cipher}, nil
}
func seal(key, plain, aad []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return gcm.Seal(nil, nonce, plain, aad), nonce, nil
}
func open(key, ciphertext, nonce, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, aad)
}
func aad(connection uuid.UUID, name string) []byte { return []byte(connection.String() + "\n" + name) }
func (s *Store) Put(ctx context.Context, connectionID, name string, value []byte) error {
	connection, err := uuid.Parse(connectionID)
	if err != nil || name == "" || len(value) == 0 {
		return errors.New("invalid secret")
	}
	ciphertext, nonce, version, err := s.cipher.Encrypt(string(aad(connection, name)), value)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO encrypted_secrets(id,connection_id,name,ciphertext,nonce,key_version) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(connection_id,name) DO UPDATE SET ciphertext=$4,nonce=$5,key_version=$6,updated_at=now()`, uuid.New(), connection, name, ciphertext, nonce, version)
	return err
}
func (s *Store) Get(ctx context.Context, connectionID, name string) ([]byte, error) {
	connection, err := uuid.Parse(connectionID)
	if err != nil {
		return nil, ErrNotFound
	}
	var ciphertext, nonce []byte
	var version int
	err = s.pool.QueryRow(ctx, `SELECT ciphertext,nonce,key_version FROM encrypted_secrets WHERE connection_id=$1 AND name=$2`, connection, name).Scan(&ciphertext, &nonce, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.cipher.Decrypt(string(aad(connection, name)), ciphertext, nonce, version)
}
func (s *Store) PutTx(ctx context.Context, tx pgx.Tx, connectionID, name string, value []byte) error {
	connection, err := uuid.Parse(connectionID)
	if err != nil || name == "" || len(value) == 0 {
		return errors.New("invalid secret")
	}
	ciphertext, nonce, version, err := s.cipher.Encrypt(string(aad(connection, name)), value)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO encrypted_secrets(id,connection_id,name,ciphertext,nonce,key_version) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(connection_id,name) DO UPDATE SET ciphertext=$4,nonce=$5,key_version=$6,updated_at=now()`, uuid.New(), connection, name, ciphertext, nonce, version)
	return err
}
