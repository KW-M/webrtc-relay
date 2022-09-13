package webrtc_relay

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/muka/peerjs-go/util"
	log "github.com/sirupsen/logrus"
)

type TokenPersistanceStore struct {
	// the Relay that this TokenPersistanceStore is associated with
	Relay *WebrtcRelay
	// Log: The logrus logger to use for debug logs within WebrtcRelay Code
	log *log.Entry
	// the map of peerIds to tokens (loaded from the json file)
	tokenMap TokenPersistanceFileJson
}

// NewTokenPersistanceStore: Creates a new TokenPersistanceStore
func NewTokenPersistanceStore(relay *WebrtcRelay) *TokenPersistanceStore {
	return &TokenPersistanceStore{
		Relay:    relay,
		log:      relay.Log.WithField("mod", "token_persistance_store"),
		tokenMap: make(map[string]string),
	}
}

// GetToken: Loads the token corresponding to the passed peerId from the json TokenPersistanceStore or creates and saves a new one if no token exists for that peer id
func (tps *TokenPersistanceStore) GetToken(peerId string) string {
	tps.log.Debug("Loading token for peer: ", peerId)
	if len(tps.tokenMap) == 0 {
		tps.log.Debug("TokenMap is empty, reading from file")
		err := tps.readJsonStore()
		if err != nil {
			tps.log.Error("Error reading token persistance store file (store file will be overwritten): ", err)
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
	if tps.Relay.config.TokenPersistanceFile == "" {
		return nil // do not load from file if no file is specified
	}
	tps.log.Debug("Reading token persistance store from file: ", tps.Relay.config.TokenPersistanceFile)
	jsonFile, err := os.Open(tps.Relay.config.TokenPersistanceFile)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
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
	if tps.Relay.config.TokenPersistanceFile == "" {
		return nil // do not write to file if no file is specified
	}
	tps.log.Debug("Writing token persistance store to file: ", tps.Relay.config.TokenPersistanceFile)
	jsonFile, err := os.Create(tps.Relay.config.TokenPersistanceFile)
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
	tps.writeJsonStore()
	return newToken
}
