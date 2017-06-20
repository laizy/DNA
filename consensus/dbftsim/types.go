package dbftsim


type TransactionType byte

const (
	BookKeeping    TransactionType = 0x00
	BookKeeper     TransactionType = 0x02
	RegisterAsset  TransactionType = 0x40
	IssueAsset     TransactionType = 0x01
	TransferAsset  TransactionType = 0x10
	Record         TransactionType = 0x11
	DeployCode     TransactionType = 0xd0
	PrivacyPayload TransactionType = 0x20
	DataFile       TransactionType = 0x12
)

type Transaction struct {
	hash int64,
	TxType      TransactionType,
}

func (self *Transaction) Hash() int64 {
	return self.hash
}
