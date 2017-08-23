chainstore 键值对
=======================

前缀：

```
	// DATA
	DATA_BlockHash   DataEntryPrefix = 0x00
	DATA_Header      DataEntryPrefix = 0x01
	DATA_Transaction DataEntryPrefix = 0x02
	DATA_Contract    DataEntryPrefix = 0x03

	// INDEX
	IX_HeaderHashList DataEntryPrefix = 0x80
	IX_Enrollment     DataEntryPrefix = 0x84
	IX_Unspent        DataEntryPrefix = 0x90
	IX_Unspent_UTXO   DataEntryPrefix = 0x91
	IX_Vote           DataEntryPrefix = 0x94

	// ASSET
	ST_Info           DataEntryPrefix = 0xc0
	ST_QuantityIssued DataEntryPrefix = 0xc1
	ST_ACCOUNT        DataEntryPrefix = 0xc2

	//SYSTEM
	SYS_CurrentBlock      DataEntryPrefix = 0x40
	SYS_CurrentHeader     DataEntryPrefix = 0x41
	SYS_CurrentBookKeeper DataEntryPrefix = 0x42

	//CONFIG
	CFG_Version DataEntryPrefix = 0xf0

```

系统变量
---------------
```
key : SYS_CurrentBlock
val : hash+uint32  
用途: 记录最后一个block的hash + block的高度

key: SYS_CurrentHeader
val: hash+uint32  
用途: 记录最后一个blockHeader的hash + 高度

key: SYS_CurrentBookKeeper
val: currBookKeeper列表 + nextBookKeeper 列表
用途: 记录bookkeeper列表信息

```

数据
-------------
```
区块hash： 
key： DATA_BlockHash + 区块高度
val: hash 
用途：通过高度查区块hash

区块头： 
key: DATA_Header + hash
val: uint64(systemfee?) + trim_block
用途：通过hash查块头。 没收到完整块前，val只是header， 收到完整块后， val是header + 交易hash列表

交易： 
key: DATA_Transaction + txhash
val: height + serialize(tx)
用途：根据hash查交易


合约：
目前只实现取，没有存的函数。
key: DATA_Transaction + hash
val: ???

```

索引
----------------
```
头hash列表
key: IX_HeaderHashList + 起始高度
val: len + hash列表
用途： 记录所有的区块hash列表，加快重新启动时的速度

IX_Enrollment: 暂时没使用


交易未花费的index列表：
key: IX_Unspent + txhash
val: 未花费的index列表 
用途： 通过交易hash 查找该交易还未用完的Output索引


某一个地址在某一个资产下的未花费列表：
key: IX_Unspent_UTXO + programhash + assetid
val:  UTXOUnspent 列表
用途： 通过programhash 和assetid 查找该地址下对应资产的UTXOUnspent 列表

IX_Vote 未实现

```

Asset
-----------------
```
资产信息：
key: ST_Info + assetId
val: serialize(asset)
用途：根据assetid查找该资产的信息

资产已发行数量:
key: ST_QuantityIssued + assetid
val: fixed64 数量
用途：根据assetid查找该资产的已发行数量

账户信息：
key: ST_ACCOUNT + programhash
val: AccountState 包含programHash, isFrozen和每个资产下的余额
用途：根据programhash查找该地址下各个资产的余额等信息

```