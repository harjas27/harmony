package block

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"io"
	"math/big"
	"reflect"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	blockif "github.com/harmony-one/harmony/block/interface"
	v0 "github.com/harmony-one/harmony/block/v0"
	v1 "github.com/harmony-one/harmony/block/v1"
	v2 "github.com/harmony-one/harmony/block/v2"
	v3 "github.com/harmony-one/harmony/block/v3"
	"github.com/harmony-one/harmony/crypto/hash"
	"github.com/harmony-one/taggedrlp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Header represents a block header in the Harmony blockchain.
type Header struct {
	blockif.Header
}

// HeaderPair ..
type HeaderPair struct {
	BeaconHeader *Header `json:"beacon-chain-header"`
	ShardHeader  *Header `json:"shard-chain-header"`
}

var (
	// ErrHeaderIsNil ..
	ErrHeaderIsNil = errors.New("cannot encode nil header receiver")
)

// MarshalJSON ..
func (h Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		N  *big.Int `json:"number"`
		H  string   `json:"hash"`
		P  string   `json:"parentHash"`
		B  string   `json:"logsBloom"`
		T  string   `json:"transactionsRoot"`
		S  string   `json:"stateRoot"`
		R  string   `json:"receiptsRoot"`
		M  string   `json:"miner"`
		E  string   `json:"extraData"`
		GL uint64   `json:"gasLimit"`
		GU uint64   `json:"gasUsed"`
		TS *big.Int `json:"timestamp"`
	}{
		h.Header.Number(),
		h.Header.Hash().Hex(),
		h.Header.ParentHash().Hex(),
		hexutil.Encode(h.Header.Bloom().Bytes()),
		h.Header.TxHash().Hex(),
		h.Header.Root().Hex(),
		h.Header.ReceiptHash().Hex(),
		h.Header.Coinbase().Hex(),
		hexutil.Encode(h.Header.Extra()),
		h.Header.GasLimit(),
		h.Header.GasUsed(),
		h.Header.Time(),
	})
}

// String ..
func (h Header) String() string {
	s, _ := json.Marshal(h)
	return string(s)
}

// EncodeRLP encodes the header using tagged RLP representation.
func (h *Header) EncodeRLP(w io.Writer) error {
	if h == nil {
		return ErrHeaderIsNil
	}
	return HeaderRegistry.Encode(w, h.Header)
}

// DecodeRLP decodes the header using tagged RLP representation.
func (h *Header) DecodeRLP(s *rlp.Stream) error {
	if h == nil {
		return ErrHeaderIsNil
	}
	decoded, err := HeaderRegistry.Decode(s)
	if err != nil {
		return err
	}
	hif, ok := decoded.(blockif.Header)
	if !ok {
		return errors.Errorf(
			"decoded object (type %s) does not implement Header interface",
			taggedrlp.TypeName(reflect.TypeOf(decoded)))
	}
	h.Header = hif
	return nil
}

// Hash returns the block hash of the header.  This uses HeaderRegistry to
// choose and return the right tagged RLP form of the header.
func (h *Header) Hash() ethcommon.Hash {
	return hash.FromRLP(h)
}

// Logger returns a sub-logger with block contexts added.
func (h *Header) Logger(logger *zerolog.Logger) *zerolog.Logger {
	nlogger := logger.With().
		Str("blockHash", h.Hash().Hex()).
		Uint32("blockShard", h.ShardID()).
		Uint64("blockEpoch", h.Epoch().Uint64()).
		Uint64("blockNumber", h.Number().Uint64()).
		Logger()
	return &nlogger
}

// With returns a field setter context for the header.
//
// Call a chain of setters on the returned field setter, followed by a call of
// Header method.  Example:
//
//	header := NewHeader(epoch).With().
//		ParentHash(parent.Hash()).
//		ShardID(parent.ShardID()).
//		Number(new(big.Int).Add(parent.Number(), big.NewInt(1)).
//		Header()
func (h *Header) With() HeaderFieldSetter {
	return HeaderFieldSetter{h: h}
}

// IsLastBlockInEpoch returns True if it is the last block of the epoch.
// Note that the last block contains the shard state of the next epoch.
func (h *Header) IsLastBlockInEpoch() bool {
	return len(h.ShardState()) > 0
}

// HeaderRegistry is the taggedrlp type registry for versioned headers.
var HeaderRegistry = taggedrlp.NewRegistry()

func init() {
	HeaderRegistry.MustRegister(taggedrlp.LegacyTag, v0.NewHeader())
	HeaderRegistry.MustAddFactory(func() interface{} { return v0.NewHeader() })
	HeaderRegistry.MustRegister("v1", v1.NewHeader())
	HeaderRegistry.MustAddFactory(func() interface{} { return v1.NewHeader() })
	HeaderRegistry.MustRegister("v2", v2.NewHeader())
	HeaderRegistry.MustAddFactory(func() interface{} { return v2.NewHeader() })
	HeaderRegistry.MustRegister("v3", v3.NewHeader())
	HeaderRegistry.MustAddFactory(func() interface{} { return v3.NewHeader() })
}
