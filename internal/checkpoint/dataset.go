package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	val "suitop/internal/validator"
)

type validatorEntry struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Signed  uint64 `json:"signed"`
	Total   uint64 `json:"total"`
	Bitmap  []byte `json:"bitmap"`
}

type epochData struct {
	Epoch           uint64                     `json:"epoch"`
	StartCheckpoint uint64                     `json:"start_checkpoint"`
	EndCheckpoint   uint64                     `json:"end_checkpoint"`
	Validators      map[string]*validatorEntry `json:"-"`
	Order           []string                   `json:"-"`
}

// DatasetManager manages dataset collection for a single epoch.
type DatasetManager struct {
	folder string
	data   *epochData
}

func NewDatasetManager(folder string) (*DatasetManager, error) {
	if folder == "" {
		folder = "./data"
	}
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, err
	}
	return &DatasetManager{folder: folder}, nil
}

func (dm *DatasetManager) startEpoch(epoch uint64, committee []val.ValidatorInfo, startSeq uint64) {
	dm.data = &epochData{
		Epoch:           epoch,
		StartCheckpoint: startSeq,
		EndCheckpoint:   startSeq,
		Validators:      make(map[string]*validatorEntry),
		Order:           make([]string, 0, len(committee)),
	}
	for _, v := range committee {
		dm.data.Validators[v.SuiAddress] = &validatorEntry{Name: v.Name, Address: v.SuiAddress}
		dm.data.Order = append(dm.data.Order, v.SuiAddress)
	}
}

func (dm *DatasetManager) appendBit(v *validatorEntry, signed bool) {
	byteIndex := int(v.Total / 8)
	if byteIndex >= len(v.Bitmap) {
		v.Bitmap = append(v.Bitmap, 0)
	}
	if signed {
		v.Bitmap[byteIndex] |= 1 << (v.Total % 8)
		v.Signed++
	}
	v.Total++
}

// RecordCheckpoint records signatures for a checkpoint.
func (dm *DatasetManager) RecordCheckpoint(epoch uint64, seq uint64, bitmap []uint32, committee []val.ValidatorInfo) {
	if dm.data == nil {
		dm.startEpoch(epoch, committee, seq)
	}
	if dm.data.Epoch != epoch {
		dm.finishEpoch()
		dm.startEpoch(epoch, committee, seq)
	}
	dm.data.EndCheckpoint = seq
	for _, v := range committee {
		entry, ok := dm.data.Validators[v.SuiAddress]
		if !ok {
			entry = &validatorEntry{Name: v.Name, Address: v.SuiAddress}
			dm.data.Validators[v.SuiAddress] = entry
			dm.data.Order = append(dm.data.Order, v.SuiAddress)
		}
		signed := IsValidatorSigned(bitmap, v.BitmapIndex)
		dm.appendBit(entry, signed)
	}
}

func (dm *DatasetManager) finishEpoch() {
	if dm.data == nil {
		return
	}
	fileName := fmt.Sprintf("epoch_%d_%d-%d.json", dm.data.Epoch, dm.data.StartCheckpoint, dm.data.EndCheckpoint)
	path := fmt.Sprintf("%s/%s", dm.folder, fileName)

	out := struct {
		Epoch           uint64           `json:"epoch"`
		StartCheckpoint uint64           `json:"start_checkpoint"`
		EndCheckpoint   uint64           `json:"end_checkpoint"`
		Validators      []validatorEntry `json:"validators"`
	}{
		Epoch:           dm.data.Epoch,
		StartCheckpoint: dm.data.StartCheckpoint,
		EndCheckpoint:   dm.data.EndCheckpoint,
	}
	for _, addr := range dm.data.Order {
		out.Validators = append(out.Validators, *dm.data.Validators[addr])
	}

	f, err := os.Create(path)
	if err == nil {
		json.NewEncoder(f).Encode(out)
		f.Close()
	}
	dm.data = nil
}

func (dm *DatasetManager) Close() {
	dm.finishEpoch()
}
