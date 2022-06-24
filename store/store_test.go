package store

import (
	"reflect"
	"testing"
	"time"

	"orblivion/lbry-id/auth"
	"orblivion/lbry-id/wallet"
)

func expectTokenExists(t *testing.T, s *Store, token auth.TokenString, expectedToken auth.AuthToken) {
	gotToken, err := s.GetToken(token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, expectedToken) {
		t.Fatalf("token: \n  expected %+v\n  got:     %+v", expectedToken, *gotToken)
	}
}

func expectTokenNotExists(t *testing.T, s *Store, token auth.TokenString) {
	gotToken, err := s.GetToken(token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}
}

// Test insertToken, using GetToken as a helper
// Try insertToken twice with the same user and device, error the second time
func TestStoreInsertToken(t *testing.T) {

	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// created for addition to the DB (no expiration attached)
	authToken1 := auth.AuthToken{
		Token:    "seekrit-1",
		DeviceId: "dId",
		Scope:    "*",
		UserId:   123,
	}
	expiration := time.Now().Add(time.Hour * 24 * 14).UTC()

	// Get a token, come back empty
	expectTokenNotExists(t, &s, authToken1.Token)

	// Put in a token
	if err := s.insertToken(&authToken1, expiration); err != nil {
		t.Fatalf("Unexpected error in insertToken: %+v", err)
	}

	// The value expected when we pull it from the database.
	authToken1Expected := authToken1
	authToken1Expected.Expiration = &expiration

	// Get and confirm the token we just put in
	expectTokenExists(t, &s, authToken1.Token, authToken1Expected)

	// Try to put a different token, fail because we already have one
	authToken2 := authToken1
	authToken2.Token = "seekrit-2"

	if err := s.insertToken(&authToken2, expiration); err != ErrDuplicateToken {
		t.Fatalf(`insertToken err: wanted "%+v", got "%+v"`, ErrDuplicateToken, err)
	}

	// Get the same *first* token we successfully put in
	expectTokenExists(t, &s, authToken1.Token, authToken1Expected)
}

// Test updateToken, using GetToken and insertToken as helpers
// Try updateToken with no existing token, err for lack of anything to update
// Try updateToken with a preexisting token, succeed
// Try updateToken again with a new token, succeed
func TestStoreUpdateToken(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// created for addition to the DB (no expiration attached)
	authTokenUpdate := auth.AuthToken{
		Token:    "seekrit-update",
		DeviceId: "dId",
		Scope:    "*",
		UserId:   123,
	}
	expiration := time.Now().Add(time.Hour * 24 * 14).UTC()

	// Try to get a token, come back empty because we're just starting out
	expectTokenNotExists(t, &s, authTokenUpdate.Token)

	// Try to update the token - fail because we don't have an entry there in the first place
	if err := s.updateToken(&authTokenUpdate, expiration); err != ErrNoToken {
		t.Fatalf(`updateToken err: wanted "%+v", got "%+v"`, ErrNoToken, err)
	}

	// Try to get a token, come back empty because the update attempt failed to do anything
	expectTokenNotExists(t, &s, authTokenUpdate.Token)

	// Put in a different token, just so we have something to test that
	// updateToken overwrites it
	authTokenInsert := authTokenUpdate
	authTokenInsert.Token = "seekrit-insert"

	if err := s.insertToken(&authTokenInsert, expiration); err != nil {
		t.Fatalf("Unexpected error in insertToken: %+v", err)
	}

	// Now successfully update token
	if err := s.updateToken(&authTokenUpdate, expiration); err != nil {
		t.Fatalf("Unexpected error in updateToken: %+v", err)
	}

	// The value expected when we pull it from the database.
	authTokenUpdateExpected := authTokenUpdate
	authTokenUpdateExpected.Expiration = &expiration

	// Get and confirm the token we just put in
	expectTokenExists(t, &s, authTokenUpdate.Token, authTokenUpdateExpected)

	// Fail to get the token we previously inserted, because it's now been overwritten
	expectTokenNotExists(t, &s, authTokenInsert.Token)
}

// Test that a user can have two different devices.
// Test first and second Save (one for insert, one for update)
// Get fails initially
// Put token1-d1 token1-d2
// Get token1-d1 token1-d2
// Put token2-d1 token2-d2
// Get token2-d1 token2-d2
func TestStoreSaveToken(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Version 1 of the token for both devices
	// created for addition to the DB (no expiration attached)
	authToken_d1_1 := auth.AuthToken{
		Token:    "seekrit-d1-1",
		DeviceId: "dId-1",
		Scope:    "*",
		UserId:   123,
	}

	authToken_d2_1 := authToken_d1_1
	authToken_d2_1.DeviceId = "dId-2"
	authToken_d2_1.Token = "seekrit-d2-1"

	// Try to get the tokens, come back empty because we're just starting out
	expectTokenNotExists(t, &s, authToken_d1_1.Token)
	expectTokenNotExists(t, &s, authToken_d2_1.Token)

	// Save Version 1 tokens for both devices
	if err := s.SaveToken(&authToken_d1_1); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}
	if err := s.SaveToken(&authToken_d2_1); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}

	// Check one of the authTokens to make sure expiration was set
	if authToken_d1_1.Expiration == nil {
		t.Fatalf("Expected SaveToken to set an Expiration")
	}
	nowDiff := authToken_d1_1.Expiration.Sub(time.Now())
	if time.Hour*24*14+time.Minute < nowDiff || nowDiff < time.Hour*24*14-time.Minute {
		t.Fatalf("Expected SaveToken to set a token Expiration 2 weeks in the future.")
	}

	// Get and confirm the tokens we just put in
	expectTokenExists(t, &s, authToken_d1_1.Token, authToken_d1_1)
	expectTokenExists(t, &s, authToken_d2_1.Token, authToken_d2_1)

	// Version 2 of the token for both devices
	authToken_d1_2 := authToken_d1_1
	authToken_d1_2.Token = "seekrit-d1-2"

	authToken_d2_2 := authToken_d2_1
	authToken_d2_2.Token = "seekrit-d2-2"

	// Save Version 2 tokens for both devices
	if err := s.SaveToken(&authToken_d1_2); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}
	if err := s.SaveToken(&authToken_d2_2); err != nil {
		t.Fatalf("Unexpected error in SaveToken: %+v", err)
	}

	// Check that the expiration of this new token is marginally later
	if authToken_d1_2.Expiration == nil {
		t.Fatalf("Expected SaveToken to set an Expiration")
	}
	expDiff := authToken_d1_2.Expiration.Sub(*authToken_d1_1.Expiration)
	if time.Second < expDiff || expDiff < 0 {
		t.Fatalf("Expected new expiration to be slightly later than previous expiration. diff: %+v", expDiff)
	}

	// Get and confirm the tokens we just put in
	expectTokenExists(t, &s, authToken_d1_2.Token, authToken_d1_2)
	expectTokenExists(t, &s, authToken_d2_2.Token, authToken_d2_2)
}

// test GetToken using insertToken and updateToken as helpers (so we can set expiration timestamps)
// normal
// token not found
// expired not returned
func TestStoreGetToken(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// created for addition to the DB (no expiration attached)
	authToken := auth.AuthToken{
		Token:    "seekrit-d1",
		DeviceId: "dId",
		Scope:    "*",
		UserId:   123,
	}
	expiration := time.Time(time.Now().UTC().Add(time.Hour * 24 * 14))

	// Not found (nothing saved for this token string)
	gotToken, err := s.GetToken(authToken.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken. token: %+v err: %+v", gotToken, err)
	}

	// Put in a token
	if err := s.insertToken(&authToken, expiration); err != nil {
		t.Fatalf("Unexpected error in insertToken: %+v", err)
	}

	// The value expected when we pull it from the database.
	authTokenExpected := authToken
	authTokenExpected.Expiration = &expiration

	// Confirm it saved
	gotToken, err = s.GetToken(authToken.Token)
	if err != nil {
		t.Fatalf("Unexpected error in GetToken: %+v", err)
	}
	if gotToken == nil || !reflect.DeepEqual(*gotToken, authTokenExpected) {
		t.Fatalf("token: \n  expected %+v\n  got:     %+v", authTokenExpected, gotToken)
	}

	// Update the token to be expired
	expirationOld := time.Now().Add(time.Second * (-1))
	if err := s.updateToken(&authToken, expirationOld); err != nil {
		t.Fatalf("Unexpected error in updateToken: %+v", err)
	}

	// Fail to get the expired token
	gotToken, err = s.GetToken(authToken.Token)
	if gotToken != nil || err != ErrNoToken {
		t.Fatalf("Expected ErrNoToken, for expired token. token: %+v err: %+v", gotToken, err)
	}
}

func TestStoreSanitizeEmptyFields(t *testing.T) {
	// Make sure expiration doesn't get set if sanitization fails
	t.Fatalf("Test me")
}

func TestStoreTimeZones(t *testing.T) {
	// Make sure the tz situation is as we prefer in the DB unless we just do UTC.
	t.Fatalf("Test me")
}

func expectWalletExists(
	t *testing.T,
	s *Store,
	userId auth.UserId,
	expectedEncryptedWallet wallet.EncryptedWallet,
	expectedSequence wallet.Sequence,
	expectedHmac wallet.WalletHmac,
) {
	encryptedWallet, sequence, hmac, err := s.GetWallet(userId)
	if encryptedWallet != expectedEncryptedWallet || sequence != expectedSequence || hmac != expectedHmac || err != nil {
		t.Fatalf("Unexpected values for wallet: encrypted wallet: %+v sequence: %+v hmac: %+v err: %+v", encryptedWallet, sequence, hmac, err)
	}
}

func expectWalletNotExists(t *testing.T, s *Store, userId auth.UserId) {
	encryptedWallet, sequence, hmac, err := s.GetWallet(userId)
	if len(encryptedWallet) != 0 || sequence != 0 || len(hmac) != 0 || err != ErrNoWallet {
		t.Fatalf("Expected ErrNoWallet, and no wallet values. Instead got: encrypted wallet: %+v sequence: %+v hmac: %+v err: %+v", encryptedWallet, sequence, hmac, err)
	}
}

func setupWalletTest(s *Store) auth.UserId {
	email, password := auth.Email("abc@example.com"), auth.Password("123")
	_ = s.CreateAccount(email, password)
	userId, _ := s.GetUserId(email, password)
	return userId
}

// Test insertFirstWallet, using GetWallet, CreateAccount and GetUserID as a helpers
// Try insertFirstWallet twice with the same user id, error the second time
func TestStoreInsertWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId := setupWalletTest(&s)

	// Get a wallet, come back empty
	expectWalletNotExists(t, &s, userId)

	// Put in a first wallet
	if err := s.insertFirstWallet(userId, wallet.EncryptedWallet("my-enc-wallet"), wallet.WalletHmac("my-hmac")); err != nil {
		t.Fatalf("Unexpected error in insertFirstWallet: %+v", err)
	}

	// Get a wallet, have the values we put in with a sequence of 1
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet"), wallet.Sequence(1), wallet.WalletHmac("my-hmac"))

	// Put in a first wallet for a second time, have an error for trying
	if err := s.insertFirstWallet(userId, wallet.EncryptedWallet("my-enc-wallet-2"), wallet.WalletHmac("my-hmac-2")); err != ErrDuplicateWallet {
		t.Fatalf(`insertFirstWallet err: wanted "%+v", got "%+v"`, ErrDuplicateToken, err)
	}

	// Get the same *first* wallet we successfully put in
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet"), wallet.Sequence(1), wallet.WalletHmac("my-hmac"))
}

// Test updateWalletToSequence, using GetWallet, CreateAccount, GetUserID, and insertFirstWallet as helpers
// Try updateWalletToSequence with no existing wallet, err for lack of anything to update
// Try updateWalletToSequence with a preexisting wallet but the wrong sequence, fail
// Try updateWalletToSequence with a preexisting wallet and the correct sequence, succeed
// Try updateWalletToSequence again, succeed
func TestStoreUpdateWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId := setupWalletTest(&s)

	// Try to update a wallet, fail for nothing to update
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a")); err != ErrNoWallet {
		t.Fatalf(`updateWalletToSequence err: wanted "%+v", got "%+v"`, ErrNoWallet, err)
	}

	// Get a wallet, come back empty since it failed
	expectWalletNotExists(t, &s, userId)

	// Put in a first wallet
	if err := s.insertFirstWallet(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.WalletHmac("my-hmac-a")); err != nil {
		t.Fatalf("Unexpected error in insertFirstWallet: %+v", err)
	}

	// Try to update the wallet, fail for having the wrong sequence
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-b")); err != ErrNoWallet {
		t.Fatalf(`updateWalletToSequence err: wanted "%+v", got "%+v"`, ErrNoWallet, err)
	}

	// Get the same wallet we initially *inserted*, since it didn't update
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Update the wallet successfully, with the right sequence
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b")); err != nil {
		t.Fatalf("Unexpected error in updateWalletToSequence: %+v", err)
	}

	// Get a wallet, have the values we put in
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b"))

	// Update the wallet again successfully
	if err := s.updateWalletToSequence(userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c")); err != nil {
		t.Fatalf("Unexpected error in updateWalletToSequence: %+v", err)
	}

	// Get a wallet, have the values we put in
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c"))
}

// NOTE - the "behind the scenes" comments give a view of what we're expecting
// to happen, and why we're testing what we are. Sometimes it should insert,
// sometimes it should update. It depends on whether it's the first wallet
// submitted, and that's easily determined by sequence=1. However, if we switch
// to a database with "upserts" and take advantage of it, what happens behind
// the scenes will change a little, so the comments should be updated. Though,
// we'd probably best test the same cases.
//
// TODO when we have lastSynced again: test fail via update for having
// non-matching device sequence history. Though, maybe this goes into wallet
// util
func TestStoreSetWallet(t *testing.T) {
	s, sqliteTmpFile := StoreTestInit(t)
	defer StoreTestCleanup(sqliteTmpFile)

	// Get a valid userId
	userId := setupWalletTest(&s)

	// Sequence 2 - fails - out of sequence (behind the scenes, tries to update but there's nothing there yet)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-a")); err != ErrWrongSequence {
		t.Fatalf(`SetWallet err: wanted "%+v", got "%+v"`, ErrWrongSequence, err)
	}
	expectWalletNotExists(t, &s, userId)

	// Sequence 1 - succeeds - out of sequence (behind the scenes, does an insert)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a")); err != nil {
		t.Fatalf("Unexpected error in SetWallet: %+v", err)
	}
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Sequence 1 - fails - out of sequence (behind the scenes, tries to insert but there's something there already)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-b")); err != ErrWrongSequence {
		t.Fatalf(`SetWallet err: wanted "%+v", got "%+v"`, ErrWrongSequence, err)
	}
	// Expect the *first* wallet to still be there
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Sequence 3 - fails - out of sequence (behind the scenes: tries via update, which is appropriate here)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-b")); err != ErrWrongSequence {
		t.Fatalf(`SetWallet err: wanted "%+v", got "%+v"`, ErrWrongSequence, err)
	}
	// Expect the *first* wallet to still be there
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-a"), wallet.Sequence(1), wallet.WalletHmac("my-hmac-a"))

	// Sequence 2 - succeeds - (behind the scenes, does an update. Tests successful update-after-insert)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b")); err != nil {
		t.Fatalf("Unexpected error in SetWallet: %+v", err)
	}
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-b"), wallet.Sequence(2), wallet.WalletHmac("my-hmac-b"))

	// Sequence 3 - succeeds - (behind the scenes, does an update. Tests successful update-after-update. Maybe gratuitous?)
	if err := s.SetWallet(userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c")); err != nil {
		t.Fatalf("Unexpected error in SetWallet: %+v", err)
	}
	expectWalletExists(t, &s, userId, wallet.EncryptedWallet("my-enc-wallet-c"), wallet.Sequence(3), wallet.WalletHmac("my-hmac-c"))
}

func TestStoreGetWalletSuccess(t *testing.T) {
	t.Fatalf("Test me: Wallet get success")
}

func TestStoreGetWalletFail(t *testing.T) {
	t.Fatalf("Test me: Wallet get failures")
}

func TestStoreCreateAccount(t *testing.T) {
	t.Fatalf("Test me: Account create success and failures")
}

func TestStoreGetUserId(t *testing.T) {
	t.Fatalf("Test me: User ID get success and failures")
}
