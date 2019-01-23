package did

import (
	"encoding/json"
	"errors"
	"strings"
)

type DDO struct {
	Context string            `json:"@context"`
	ID      string            `json:"id"`
	Cert    *Cert             `json:"cert,omitempty"`
	Owner   []*OwnerKey       `json:"owner"`
	Service map[string]string `json:"service,omitempty"`
	Sig     *Signature        `json:"signature,omitempty"`
}

func CreateDDO(did string, pk PubKey, sk PriKey, cert *Cert, service map[string]string) ([]byte, error) {
	ddo := DDO{
		Context: "http://example.com/context",
		ID:      did,
		Cert:    cert,
		Owner: []*OwnerKey{&OwnerKey{
			ID:  did + "#key/1",
			Key: pk,
		}},
		Service: service,
		Sig:     nil,
	}
	raw, err := json.Marshal(ddo)

	if err != nil {
		return nil, err
	}

	sig, err := sk.Sign(raw)
	if err != nil {
		return nil, err
	}

	alg := ECDSA
	switch sk.(type) {
	case DNAPriKey:
		if CurveName() == "SM2" {
			alg = SM2
		}
		break
	default:
		return nil, errors.New("unknown signature algorithm")
	}
	ddo.Sig, err = ConstructSignature(ddo.Owner[0].ID, alg, sig)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ddo)
}

func (p *DDO) VerifySignature() error {
	if p.Sig == nil {
		return errors.New("DDO does not contain a signature")
	}

	msg, err := json.Marshal(DDO{
		Context: p.Context,
		ID:      p.ID,
		Cert:    p.Cert,
		Owner:   p.Owner,
		Service: p.Service,
		Sig:     nil,
	})
	if err != nil {
		return err
	}
	//TODO assume key and signature are all in ECDSA
	for _, key := range p.Owner {
		if key.ID == p.Sig.Creator {
			if !key.Key.VerifyAddress(key.ID) {
				return errors.New("verification address/publickey failed")
			}
			if key.Key.Verify(msg, p.Sig.Value) {
				return nil
			}
			break
		}
	}
	return errors.New("verification failed")
}

func (p *DDO) GetKey(ref string) (PubKey, error) {
	i := strings.Index(ref, "#")
	if i == -1 {
		return nil, errors.New("the id reference should contain fragment of the owner's key")
	}

	id := ref[:i]
	if id != p.ID {
		return nil, errors.New("id dose not match")
	}

	for _, key := range p.Owner {
		if key.ID == ref {
			return key.Key, nil
		}
	}

	return nil, errors.New("no such fragment")
}