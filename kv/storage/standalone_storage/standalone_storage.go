package standalone_storage

import (
	"github.com/Connor1996/badger"
	"github.com/pingcap-incubator/tinykv/kv/config"
	"github.com/pingcap-incubator/tinykv/kv/storage"
	"github.com/pingcap-incubator/tinykv/kv/util/engine_util"
	"github.com/pingcap-incubator/tinykv/proto/pkg/kvrpcpb"
)

// StandAloneStorage is an implementation of `Storage` for a single-node TinyKV instance. It does not
// communicate with other nodes and all data is stored locally.
type StandAloneStorage struct {
	// Your Data Here (1).
	conf *config.Config
	db   *badger.DB
}

// NewStandAloneStorage 创建了一个新的StandAloneStorage实例
func NewStandAloneStorage(conf *config.Config) *StandAloneStorage {
	// Your Code Here (1).
	return &StandAloneStorage{
		conf: conf,
		db:   nil,
	}
}

// Start 连接数据库
func (s *StandAloneStorage) Start() error {
	// Your Code Here (1).
	s.db = engine_util.CreateDB(s.conf.DBPath, s.conf.Raft)
	return nil
}

func (s *StandAloneStorage) Stop() error {
	// Your Code Here (1).
	return nil
}

// Reader 返回一个 StorageReader 并且创建一个只读事务
func (s *StandAloneStorage) Reader(ctx *kvrpcpb.Context) (storage.StorageReader, error) {
	// Your Code Here (1).
	return &StandAloneReader{
		txn: s.db.NewTransaction(false),
	}, nil
}

// Write 功能为将键值对添加到数据库，传入的键值对value为空则删除相应的key
func (s *StandAloneStorage) Write(ctx *kvrpcpb.Context, batch []storage.Modify) error {
	// Your Code Here (1).
	for _, modify := range batch {
		value := modify.Value()
		cf := modify.Cf()
		key := modify.Key()
		if value == nil { // 如果value为空就将相应的key删除
			err := engine_util.DeleteCF(s.db, cf, key)
			if err != nil {
				return err
			}
		} else { // 如果value不为空就将键值对添加到数据库
			err := engine_util.PutCF(s.db, cf, key, value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// StandAloneReader 为了返回StorageReader而创
type StandAloneReader struct {
	txn *badger.Txn
}

// GetCF 通过cf和key获取value
func (s *StandAloneReader) GetCF(cf string, key []byte) ([]byte, error) {
	val, err := engine_util.GetCFFromTxn(s.txn, cf, key)
	if err == badger.ErrKeyNotFound { //如果要查找的key不存在
		return nil, nil
	}
	return val, nil
}

// IterCF 我暂时也不知道干啥，反正他要我就返回
func (s *StandAloneReader) IterCF(cf string) engine_util.DBIterator {
	return engine_util.NewCFIterator(cf, s.txn)
}

// Close 关闭和数据库的连接
func (s *StandAloneReader) Close() {
	s.txn.Discard()
}
