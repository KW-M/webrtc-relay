package webrtc_relay

import (
	"encoding/json"
	"os"

	"github.com/muka/peerjs-go/util"
	log "github.com/sirupsen/logrus"
)

// TokenPersistanceFileJson is the json format of the token persistance file (see WebrtcRelayConfig.TokenPersistanceFile) where every key is a peer id this relay has recently had and the value is the corresponding token first sent when establishing this relay as peer on the peerjs server.
type tokenStoreMap map[string]string

type TokenPersistanceStore struct {
	// the Relay that this TokenPersistanceStore is associated with
	tokenStoreFilePath string
	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	log *log.Entry
	// the map of peerIds to tokens (loaded from the json file)
	tokenMap tokenStoreMap
}

// NewTokenPersistanceStore: Creates a new TokenPersistanceStore
// tps.tokenStoreFilePath
func NewTokenPersistanceStore(tokenStoreFilePath string, log *log.Logger) *TokenPersistanceStore {
	return &TokenPersistanceStore{
		tokenStoreFilePath: tokenStoreFilePath,
		log:                log.WithField("mod", "token_persistance_store"),
		tokenMap:           make(map[string]string),
	}
}

// GetToken: Loads the token corresponding to the passed peerId from the json TokenPersistanceStore or creates and saves a new one if no token exists for that peer id
func (tps *TokenPersistanceStore) GetToken(peerId string) string {
	tps.log.Debug("Loading token for peer: ", peerId)
	if len(tps.tokenMap) == 0 {
		tps.log.Debug("TokenMap is empty, reading from file")
		err := tps.readJsonStore()
		if err != nil {
			tps.log.Error("Error reading token persistance store file (store file will be overwritten): ", err.Error())
			return tps.newToken(peerId)
		}
	}

	token, ok := tps.tokenMap[peerId]
	if !ok {
		token = tps.newToken(peerId)
		tps.log.Debugf("No existing token found for peer: %s, creating one: %s", peerId, token)
		return token
	}

	tps.log.Debugf("Found token for peer %s: %s", peerId, token)
	return token
}

func (tps *TokenPersistanceStore) DiscardToken(peerId string) error {
	if _, ok := tps.tokenMap[peerId]; ok {
		delete(tps.tokenMap, peerId)
		tps.log.Debug("Discarded token for peerId: ", peerId)
		return tps.writeJsonStore()
	} else {
		return nil
	}
}

// readJsonStore: Reads the json TokenPersistanceStore from the file specified in the config
func (tps *TokenPersistanceStore) readJsonStore() error {
	if tps.tokenStoreFilePath == "" {
		return nil // do not load from file if no file is specified
	}
	tps.log.Debug("Reading token persistance store from file: ", tps.tokenStoreFilePath)
	byteValue, err := os.ReadFile(tps.tokenStoreFilePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(byteValue, &tps.tokenMap)
	if err != nil {
		return err
	}

	return nil
}

// writeJsonStore: Writes the json TokenPersistanceStore to the file specified in the config
func (tps *TokenPersistanceStore) writeJsonStore() error {
	if tps.tokenStoreFilePath == "" {
		return nil // do not write to file if no file is specified
	}
	tps.log.Debug("Writing token persistance store to file: ", tps.tokenStoreFilePath)
	jsonFile, err := os.Create(tps.tokenStoreFilePath)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	jsonBytes, err := json.MarshalIndent(tps.tokenMap, "", "  ")
	if err != nil {
		return err
	}

	_, err = jsonFile.Write(jsonBytes)
	if err != nil {
		return err
	}

	return nil
}

func (tps *TokenPersistanceStore) newToken(peerId string) string {
	newToken := util.RandomToken()
	tps.tokenMap[peerId] = newToken
	tps.log.Debug("Created new token for peerId: ", peerId)
	if err := tps.writeJsonStore(); err != nil {
		tps.log.Error("Error writing token persistance store file: ", err.Error())
	}
	return newToken
}
