package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys-ext/http/api"
	"github.com/keys-pub/keys/ds"
	"github.com/stretchr/testify/require"
)

func TestVault(t *testing.T) {
	env := newEnv(t)
	// env.logLevel = server.DebugLevel
	// keys.SetLogger(keys.NewLogger(keys.DebugLevel))

	srv := newTestServer(t, env)
	clock := env.clock

	alice := keys.NewEdX25519KeyFromSeed(keys.Bytes32(bytes.Repeat([]byte{0x01}, 32)))

	// GET /vault/:kid (not found)
	req, err := api.NewRequest("GET", ds.Path("vault", alice.ID()), nil, clock.Now(), alice)
	require.NoError(t, err)
	code, _, body := srv.Serve(req)
	require.Equal(t, http.StatusNotFound, code)
	require.Equal(t, `{"error":{"code":404,"message":"vault not found"}}`, body)

	// POST /vault/:kid/id1
	req, err = api.NewRequest("POST", ds.Path("vault", alice.ID()), bytes.NewReader([]byte("test1")), clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "{}", body)

	// GET /vault/:kid
	req, err = api.NewRequest("GET", ds.Path("vault", alice.ID()), nil, clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	var resp api.VaultResponse
	err = json.Unmarshal([]byte(body), &resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Version)
	require.Equal(t, 1, len(resp.Boxes))
	require.Equal(t, []byte("test1"), resp.Boxes[0].Data)

	// GET /vault/:kid?version=next
	req, err = api.NewRequest("GET", ds.Path("vault", alice.ID())+"?version="+resp.Version, nil, clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	var resp2 api.MessagesResponse
	err = json.Unmarshal([]byte(body), &resp2)
	require.NoError(t, err)
	require.Equal(t, 0, len(resp2.Messages))
	require.Equal(t, resp.Version, resp2.Version)

	// POST /vault/:kid
	req, err = api.NewRequest("POST", ds.Path("vault", alice.ID()), bytes.NewReader([]byte("test2")), clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, `{}`, body)
	// POST /vault/:kid
	req, err = api.NewRequest("POST", ds.Path("vault", alice.ID()), bytes.NewReader([]byte("test3")), clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, `{}`, body)

	// GET /vault/:kid?version=next
	req, err = api.NewRequest("GET", ds.Path("vault", alice.ID())+"?version="+resp.Version, nil, clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	var resp3 api.VaultResponse
	err = json.Unmarshal([]byte(body), &resp3)
	require.NoError(t, err)
	require.Equal(t, 2, len(resp3.Boxes))
	require.Equal(t, []byte("test2"), resp3.Boxes[0].Data)
	require.Equal(t, []byte("test3"), resp3.Boxes[1].Data)

	// PUT /vault/:kid
	vault := []*api.VaultBox{
		&api.VaultBox{Data: []byte("test4")},
		&api.VaultBox{Data: []byte("test5")},
		&api.VaultBox{Data: []byte("test6")},
		&api.VaultBox{Data: []byte("test7")},
		&api.VaultBox{Data: []byte("test8")},
		&api.VaultBox{Data: []byte("test9")},
	}
	data, err := json.Marshal(vault)
	require.NoError(t, err)
	req, err = api.NewRequest("PUT", ds.Path("vault", alice.ID()), bytes.NewReader(data), clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, `{}`, body)

	// GET /vault/:kid?version=next
	req, err = api.NewRequest("GET", ds.Path("vault", alice.ID())+"?version="+resp3.Version, nil, clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusOK, code)
	var resp4 api.VaultResponse
	err = json.Unmarshal([]byte(body), &resp4)
	require.NoError(t, err)
	require.Equal(t, 6, len(resp4.Boxes))
	require.Equal(t, []byte("test4"), resp4.Boxes[0].Data)
	require.Equal(t, []byte("test5"), resp4.Boxes[1].Data)
	require.Equal(t, []byte("test6"), resp4.Boxes[2].Data)
	require.Equal(t, []byte("test7"), resp4.Boxes[3].Data)
	require.Equal(t, []byte("test8"), resp4.Boxes[4].Data)
	require.Equal(t, []byte("test9"), resp4.Boxes[5].Data)
}

func TestVaultAuth(t *testing.T) {
	env := newEnv(t)
	srv := newTestServer(t, env)
	clock := env.clock

	alice := keys.NewEdX25519KeyFromSeed(keys.Bytes32(bytes.Repeat([]byte{0x01}, 32)))
	randKey := keys.GenerateEdX25519Key()

	// GET /vault/:kid (no auth)
	req, err := http.NewRequest("GET", ds.Path("vault", alice.ID()), nil)
	require.NoError(t, err)
	code, _, body := srv.Serve(req)
	require.Equal(t, http.StatusUnauthorized, code)
	require.Equal(t, `{"error":{"code":401,"message":"missing Authorization header"}}`, body)

	// GET /vault/:kid (invalid key)
	req, err = api.NewRequest("GET", ds.Path("vault", alice.ID()), nil, clock.Now(), randKey)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, `{"error":{"code":403,"message":"invalid kid"}}`, body)

	// POST /vault/:kid/id1/1 (invalid key)
	req, err = api.NewRequest("POST", ds.Path("vault", alice.ID()), bytes.NewReader([]byte("test")), clock.Now(), randKey)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, `{"error":{"code":403,"message":"invalid kid"}}`, body)

	// GET /vault/:kid
	req, err = api.NewRequest("GET", ds.Path("vault", alice.ID()), nil, clock.Now(), alice)
	require.NoError(t, err)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusNotFound, code)
	require.Equal(t, `{"error":{"code":404,"message":"vault not found"}}`, body)

	// Replay last request
	reqReplay, err := http.NewRequest("GET", req.URL.String(), nil)
	reqReplay.Header.Set("Authorization", req.Header.Get("Authorization"))
	require.NoError(t, err)
	code, _, body = srv.Serve(reqReplay)
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, `{"error":{"code":403,"message":"nonce collision"}}`, body)

	// GET /vault/:kid (invalid authorization)
	authHeader := req.Header.Get("Authorization")
	sig := strings.Split(authHeader, ":")[1]
	req, err = api.NewRequest("GET", ds.Path("vault", alice.ID()), nil, clock.Now(), randKey)
	require.NoError(t, err)
	req.Header.Set("Authorization", randKey.ID().String()+":"+sig)
	code, _, body = srv.Serve(req)
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, `{"error":{"code":403,"message":"verify failed"}}`, body)
}