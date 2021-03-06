package exception

import . "github.com/eosspark/eos-go/log"

type MongoDbException struct{ LogMessage }

func (MongoDbException) ChainExceptions()   {}
func (MongoDbException) MongoDbExceptions() {}
func (MongoDbException) Code() ExcTypes     { return 3220000 }
func (MongoDbException) What() string {
	return "Mongo DB exception"
}

type MongoDbInsertFail struct{ LogMessage }

func (MongoDbInsertFail) ChainExceptions()   {}
func (MongoDbInsertFail) MongoDbExceptions() {}
func (MongoDbInsertFail) Code() ExcTypes     { return 3220001 }
func (MongoDbInsertFail) What() string {
	return "Fail to insert new data to Mongo DB"
}

type MongoDbUpdateFail struct{ LogMessage }

func (MongoDbUpdateFail) ChainExceptions()   {}
func (MongoDbUpdateFail) MongoDbExceptions() {}
func (MongoDbUpdateFail) Code() ExcTypes     { return 3220002 }
func (MongoDbUpdateFail) What() string {
	return "Fail to update existing data in Mongo DB"
}
