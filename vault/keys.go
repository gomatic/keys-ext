package vault

import (
	"bytes"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v4"
)

const keyItemType = "key"

func newItemForKey(key *api.Key) (*Item, error) {
	if key.ID == "" {
		return nil, errors.Errorf("no secret id")
	}
	b, err := marshalKey(key)
	if err != nil {
		return nil, err
	}
	item := NewItem(key.ID.String(), b, keyItemType, key.CreatedAt)
	return item, nil
}

func marshalKey(key *api.Key) ([]byte, error) {
	return msgpack.Marshal(key)
}

// Key from vault.
func (v *Vault) Key(id keys.ID) (*api.Key, error) {
	item, err := v.Get(id.String())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return item.Key()
}

// SaveKey saves key to vault.
func (v *Vault) SaveKey(key *api.Key) (*api.Key, bool, error) {
	if key == nil {
		return nil, false, errors.Errorf("nil secret")
	}

	if key.ID == "" {
		return nil, false, errors.Errorf("no key id")
	}

	item, err := v.Get(key.ID.String())
	if err != nil {
		return nil, false, err
	}

	updated := false
	if item != nil {
		key.UpdatedAt = v.Now()
		b, err := marshalKey(key)
		if err != nil {
			return nil, false, err
		}
		item.Data = b
		if err := v.Set(item); err != nil {
			return nil, false, err
		}
		updated = true
	} else {
		now := v.Now()
		key.CreatedAt = now
		key.UpdatedAt = now

		item, err := newItemForKey(key)
		if err != nil {
			return nil, false, err
		}
		if err := v.Set(item); err != nil {
			return nil, false, err
		}
	}

	return key, updated, nil
}

// Key for Item or nil if not a recognized key type.
func (i *Item) Key() (*api.Key, error) {
	switch i.Type {
	case keyItemType:
		return unmarshalKey(i.Data)
	// Keys used to be stored as item data directly instead of as a marshaled vault.Key.
	case string(keys.X25519), string(keys.X25519Public), string(keys.EdX25519), string(keys.EdX25519Public):
		return &api.Key{
			ID:        keys.ID(i.ID),
			Data:      i.Data,
			Type:      i.Type,
			CreatedAt: i.CreatedAt,
			UpdatedAt: i.CreatedAt,
		}, nil
	default:
		return nil, nil
	}
}

func unmarshalKey(b []byte) (*api.Key, error) {
	var key api.Key
	if err := msgpack.Unmarshal(b, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// Keys returns keys from the vault.
func (v *Vault) Keys() ([]*api.Key, error) {
	items, err := v.Items()
	if err != nil {
		return nil, err
	}
	out := make([]*api.Key, 0, len(items))
	for _, i := range items {
		key, err := i.Key()
		if err != nil {
			return nil, err
		}
		if key == nil {
			continue
		}
		out = append(out, key)
	}
	return out, nil
}

// ImportSaltpack imports key into the vault from a Saltpack message.
func (v *Vault) ImportSaltpack(msg string, password string, isHTML bool) (*api.Key, error) {
	key, err := api.DecryptKeyWithPassword(msg, password)
	if err != nil {
		return nil, err
	}
	now := v.Now()
	if key.CreatedAt.IsZero() {
		key.CreatedAt = now
	}
	if key.UpdatedAt.IsZero() {
		key.UpdatedAt = now
	}
	out, _, err := v.SaveKey(key)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ExportSaltpack exports a Key from the vault to a Saltpack message.
func (v *Vault) ExportSaltpack(id keys.ID, password string) (string, error) {
	item, err := v.Get(id.String())
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", keys.NewErrNotFound(id.String())
	}
	key, err := item.Key()
	if err != nil {
		return "", err
	}
	return api.EncryptKeyWithPassword(key, password)
}

// EdX25519Keys implements wormhole.Keyring.
func (v *Vault) EdX25519Keys() ([]*keys.EdX25519Key, error) {
	ks, err := v.Keys()
	if err != nil {
		return nil, err
	}
	out := make([]*keys.EdX25519Key, 0, len(ks))
	for _, key := range ks {
		sk, err := key.AsEdX25519()
		if err != nil {
			return nil, err
		}
		if sk == nil {
			continue
		}
		out = append(out, sk)
	}
	return out, nil
}

// X25519Keys implements saltpack.Keyring.
func (v *Vault) X25519Keys() ([]*keys.X25519Key, error) {
	ks, err := v.Keys()
	if err != nil {
		return nil, err
	}
	out := make([]*keys.X25519Key, 0, len(ks))
	for _, key := range ks {
		bk, err := key.AsX25519()
		if err != nil {
			return nil, err
		}
		if bk == nil {
			continue
		}
		out = append(out, bk)
	}
	return out, nil
}

// FindEdX25519PublicKey searches all our EdX25519 public keys for a match to a converted
// X25519 public key.
func (v *Vault) FindEdX25519PublicKey(kid keys.ID) (*keys.EdX25519PublicKey, error) {
	logger.Debugf("Finding edx25519 key from an x25519 key %s", kid)
	if !kid.IsX25519() {
		return nil, errors.Errorf("not an x25519 key")
	}
	bpk, err := keys.NewX25519PublicKeyFromID(kid)
	if err != nil {
		return nil, err
	}
	spks, err := v.EdX25519PublicKeys()
	if err != nil {
		return nil, err
	}
	for _, spk := range spks {
		if bytes.Equal(spk.X25519PublicKey().Bytes(), bpk.Bytes()) {
			logger.Debugf("Found ed25519 key %s", spk.ID())
			return spk, nil
		}
	}
	logger.Debugf("EdX25519 public key not found (for X25519 public key)")
	return nil, err
}

// EdX25519PublicKeys from the vault.
// Includes public keys of EdX25519Key's.
func (v *Vault) EdX25519PublicKeys() ([]*keys.EdX25519PublicKey, error) {
	ks, err := v.Keys()
	if err != nil {
		return nil, err
	}
	out := make([]*keys.EdX25519PublicKey, 0, len(ks))
	for _, key := range ks {
		pk, err := key.AsEdX25519Public()
		if err != nil {
			return nil, err
		}
		if pk == nil {
			continue
		}
		out = append(out, pk)
	}
	return out, nil
}

// EdX25519Key ...
func (v *Vault) EdX25519Key(kid keys.ID) (*keys.EdX25519Key, error) {
	key, err := v.Key(kid)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, nil
	}
	return key.AsEdX25519()
}
