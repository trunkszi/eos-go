package chain

import (
	//"github.com/eosspark/eos-go/chain/types"
	"github.com/eosspark/eos-go/common"
	"github.com/eosspark/eos-go/entity"
	. "github.com/eosspark/eos-go/exception"
	. "github.com/eosspark/eos-go/exception/try"
)

type Idx64 struct {
	context  *ApplyContext
	itrCache *iteratorCache
}

func NewIdx64(c *ApplyContext) *Idx64 {
	return &Idx64{
		context:  c,
		itrCache: NewIteratorCache(),
	}
}

func (i *Idx64) store(scope uint64, table uint64, payer uint64, id uint64, secondary *uint64) int {

	EosAssert(common.AccountName(payer) != common.AccountName(0), &InvalidTablePayer{}, "must specify a valid account to pay for new record")
	tab := i.context.FindOrCreateTable(uint64(i.context.Receiver), scope, table, payer)

	obj := entity.SecondaryObjectI64{
		TId:          tab.ID,
		PrimaryKey:   uint64(id),
		SecondaryKey: *secondary,
		Payer:        common.AccountName(payer),
	}

	i.context.DB.Insert(&obj)
	i.context.DB.Modify(tab, func(t *entity.TableIdObject) {
		t.Count++
	})

	i.context.UpdateDbUsage(common.AccountName(payer), int64(common.BillableSizeV("index64_object")))

	i.itrCache.cacheTable(tab)
	iteratorOut := i.itrCache.add(&obj)
	i.context.ilog.Info("object:%v iteratorOut:%d code:%v scope:%v table:%v payer:%v id:%d secondary:%v",
		obj, iteratorOut, i.context.Receiver, common.ScopeName(scope), common.TableName(table), common.AccountName(payer), id, *secondary)
	return iteratorOut

}

func (i *Idx64) remove(iterator int) {

	obj := (i.itrCache.get(iterator)).(*entity.SecondaryObjectI64)
	i.context.UpdateDbUsage(obj.Payer, -int64(common.BillableSizeV("index64_object")))

	tab := i.itrCache.getTable(obj.TId)
	EosAssert(tab.Code == i.context.Receiver, &TableAccessViolation{}, "db access violation")

	i.context.DB.Modify(tab, func(t *entity.TableIdObject) {
		t.Count--
	})

	i.context.ilog.Info("object:%v iterator:%d ", *obj, iterator)

	i.context.DB.Remove(obj)
	if tab.Count == 0 {
		i.context.DB.Remove(tab)
	}
	i.itrCache.remove(iterator)

}

func (i *Idx64) update(iterator int, payer uint64, secondary *uint64) {

	obj := (i.itrCache.get(iterator)).(*entity.SecondaryObjectI64)
	objTable := i.itrCache.getTable(obj.TId)
	i.context.ilog.Info("object:%v iterator:%d payer:%v secondary:%d", *obj, iterator, common.AccountName(payer), *secondary)
	EosAssert(objTable.Code == i.context.Receiver, &TableAccessViolation{}, "db access violation")

	accountPayer := common.AccountName(payer)
	if accountPayer == common.AccountName(0) {
		accountPayer = obj.Payer
	}

	billingSize := int64(common.BillableSizeV("index64_object"))
	if obj.Payer != accountPayer {
		i.context.UpdateDbUsage(obj.Payer, -billingSize)
		i.context.UpdateDbUsage(accountPayer, +billingSize)
	}

	i.context.DB.Modify(obj, func(o *entity.SecondaryObjectI64) {
		o.SecondaryKey = *secondary
		o.Payer = accountPayer
	})

}

func (i *Idx64) findSecondary(code uint64, scope uint64, table uint64, secondary *uint64, primary *uint64) int {

	tab := i.context.FindTable(code, scope, table)
	if tab == nil {
		return -1
	}

	tableEndItr := i.itrCache.cacheTable(tab)

	obj := entity.SecondaryObjectI64{TId: tab.ID, SecondaryKey: *secondary}
	err := i.context.DB.Find("bySecondary", obj, &obj)

	if err != nil {
		return tableEndItr
	}

	*primary = obj.PrimaryKey
	iteratorOut := i.itrCache.add(&obj)
	i.context.ilog.Info("object:%v iteratorOut:%d code:%v scope:%v table:%v secondary:%d",
		obj, iteratorOut, common.AccountName(code), common.ScopeName(scope), common.TableName(table), *secondary)
	return iteratorOut
}

func (i *Idx64) lowerbound(code uint64, scope uint64, table uint64, secondary *uint64, primary *uint64) int {

	tab := i.context.FindTable(code, scope, table)
	if tab == nil {
		return -1
	}

	tableEndItr := i.itrCache.cacheTable(tab)

	obj := entity.SecondaryObjectI64{TId: tab.ID, SecondaryKey: *secondary}

	idx, _ := i.context.DB.GetIndex("bySecondary", &obj)
	itr, _ := idx.LowerBound(&obj)
	if idx.CompareEnd(itr) {
		return tableEndItr
	}

	objLowerbound := entity.SecondaryObjectI64{}
	itr.Data(&objLowerbound)
	if objLowerbound.TId != tab.ID {
		return tableEndItr
	}

	*primary = objLowerbound.PrimaryKey
	*secondary = objLowerbound.SecondaryKey

	iteratorOut := i.itrCache.add(&objLowerbound)
	i.context.ilog.Info("object:%v iteratorOut:%d code:%v scope:%v table:%v secondary:%d",
		objLowerbound, iteratorOut, common.AccountName(code), common.ScopeName(scope), common.TableName(table), secondary)
	return iteratorOut
}

func (i *Idx64) upperbound(code uint64, scope uint64, table uint64, secondary *uint64, primary *uint64) int {

	tab := i.context.FindTable(code, scope, table)
	if tab == nil {
		return -1
	}

	tableEndItr := i.itrCache.cacheTable(tab)

	obj := entity.SecondaryObjectI64{TId: tab.ID, SecondaryKey: *secondary}

	idx, _ := i.context.DB.GetIndex("bySecondary", &obj)
	itr, _ := idx.UpperBound(&obj)
	if idx.CompareEnd(itr) {
		return tableEndItr
	}

	objUpperbound := entity.SecondaryObjectI64{}
	itr.Data(&objUpperbound)
	if objUpperbound.TId != tab.ID {
		return tableEndItr
	}

	*primary = objUpperbound.PrimaryKey
	*secondary = objUpperbound.SecondaryKey

	iteratorOut := i.itrCache.add(&objUpperbound)
	i.context.ilog.Info("object:%v iteratorOut:%d code:%v scope:%v table:%v secondary:%d",
		objUpperbound, iteratorOut, common.AccountName(code), common.ScopeName(scope), common.TableName(table), secondary)
	return iteratorOut
}

func (i *Idx64) end(code uint64, scope uint64, table uint64) int {

	i.context.ilog.Info("code:%v scope:%v table:%v ",
		common.AccountName(code), common.ScopeName(scope), common.TableName(table))

	tab := i.context.FindTable(code, scope, table)
	if tab == nil {
		return -1
	}
	return i.itrCache.cacheTable(tab)
}

func (i *Idx64) next(iterator int, primary *uint64) int {

	if iterator < -1 {
		return -1
	}
	obj := (i.itrCache.get(iterator)).(*entity.SecondaryObjectI64)

	idx, _ := i.context.DB.GetIndex("bySecondary", obj)
	itr := idx.IteratorTo(obj)

	itr.Next()
	objNext := entity.SecondaryObjectI64{}
	itr.Data(&objNext)

	i.context.ilog.Info("Idx64 objNext:%v", objNext)

	if idx.CompareEnd(itr) || objNext.TId != obj.TId {
		return i.itrCache.getEndIteratorByTableID(obj.TId)
	}

	*primary = objNext.PrimaryKey

	iteratorOut := i.itrCache.add(&objNext)
	i.context.ilog.Info("object:%v iteratorIn:%d iteratorOut:%d", objNext, iterator, iteratorOut)
	return iteratorOut

}

func (i *Idx64) previous(iterator int, primary *uint64) int {

	idx, _ := i.context.DB.GetIndex("bySecondary", &entity.SecondaryObjectI64{})

	if iterator < -1 {
		tab := i.itrCache.findTablebyEndIterator(iterator)
		EosAssert(tab != nil, &InvalidTableTterator{}, "not a valid end iterator")

		objTId := entity.SecondaryObjectI64{TId: tab.ID}

		itr, _ := idx.UpperBound(&objTId)
		if idx.CompareIterator(idx.Begin(), idx.End()) || idx.CompareBegin(itr) {
			i.context.ilog.Info("iterator is the begin(nil) of index, iteratorIn:%d iteratorOut:%d", iterator, -1)
			return -1
		}

		itr.Prev()
		objPrev := entity.KeyValueObject{}
		itr.Data(&objPrev)

		if objPrev.TId != tab.ID {
			i.context.ilog.Info("previous iterator out of tid, iteratorIn:%d iteratorOut:%d", iterator, -1)
			return -1
		}

		*primary = objPrev.PrimaryKey
		iteratorOut := i.itrCache.add(&objPrev)
		i.context.ilog.Info("object:%v iteratorIn:%d iteratorOut:%d", objPrev, iterator, iteratorOut)
		return iteratorOut
	}

	obj := (i.itrCache.get(iterator)).(*entity.SecondaryObjectI64)
	itr := idx.IteratorTo(obj)

	if idx.CompareBegin(itr) {
		return -1
	}

	itr.Prev()
	objPrev := entity.SecondaryObjectI64{}
	itr.Data(&objPrev)
	i.context.ilog.Info("Idx64 objPrev:%v", objPrev)
	if objPrev.TId != obj.TId {
		return -1
	}
	*primary = objPrev.PrimaryKey

	iteratorOut := i.itrCache.add(&objPrev)
	i.context.ilog.Info("object:%v iteratorIn:%d iteratorOut:%d", objPrev, iterator, iteratorOut)
	return iteratorOut
}

func (i *Idx64) findPrimary(code uint64, scope uint64, table uint64, secondary *uint64, primary uint64) int {

	tab := i.context.FindTable(code, scope, table)
	if tab == nil {
		return -1
	}

	tableEndItr := i.itrCache.cacheTable(tab)

	obj := entity.SecondaryObjectI64{TId: tab.ID, PrimaryKey: primary}
	err := i.context.DB.Find("byPrimary", obj, &obj)
	if err != nil {
		return tableEndItr
	}

	*secondary = obj.SecondaryKey

	iteratorOut := i.itrCache.add(&obj)
	i.context.ilog.Info("object:%v iteratorOut:%d code:%v scope:%v table:%v secondary:%d primary:%d ",
		obj, iteratorOut, common.AccountName(code), common.ScopeName(scope), common.TableName(table), *secondary, primary)
	return iteratorOut
}

func (i *Idx64) lowerboundPrimary(code uint64, scope uint64, table uint64, primary *uint64) int {

	tab := i.context.FindTable(code, scope, table)
	if tab == nil {
		return -1
	}

	tableEndItr := i.itrCache.cacheTable(tab)

	obj := entity.SecondaryObjectI64{TId: tab.ID, PrimaryKey: *primary}
	idx, _ := i.context.DB.GetIndex("byPrimary", &obj)

	itr, _ := idx.LowerBound(&obj)
	if idx.CompareEnd(itr) {
		return tableEndItr
	}

	objLowerbound := entity.SecondaryObjectI64{}
	itr.Data(&objLowerbound)

	if objLowerbound.TId != tab.ID {
		return tableEndItr
	}

	return i.itrCache.add(&objLowerbound)
}

func (i *Idx64) upperboundPrimary(code uint64, scope uint64, table uint64, primary *uint64) int {

	tab := i.context.FindTable(code, scope, table)
	if tab == nil {
		return -1
	}

	tableEndItr := i.itrCache.cacheTable(tab)

	obj := entity.SecondaryObjectI64{TId: tab.ID, PrimaryKey: *primary}
	idx, _ := i.context.DB.GetIndex("byPrimary", &obj)
	itr, _ := idx.UpperBound(&obj)
	if idx.CompareEnd(itr) {
		return tableEndItr
	}
	//objUpperbound := (*types.SecondaryObjectI64)(itr.GetObject())
	objUpperbound := entity.SecondaryObjectI64{}
	itr.Data(&objUpperbound)
	if objUpperbound.TId != tab.ID {
		return tableEndItr
	}

	i.itrCache.cacheTable(tab)
	return i.itrCache.add(&objUpperbound)
}

func (i *Idx64) nextPrimary(iterator int, primary *uint64) int {

	if iterator < -1 {
		return -1
	}
	obj := (i.itrCache.get(iterator)).(*entity.SecondaryObjectI64)
	idx, _ := i.context.DB.GetIndex("byPrimary", obj)

	itr := idx.IteratorTo(obj)

	itr.Next()
	objNext := entity.SecondaryObjectI64{}
	itr.Data(&objNext)

	if idx.CompareEnd(itr) || objNext.TId != obj.TId {
		return i.itrCache.getEndIteratorByTableID(obj.TId)
	}

	*primary = objNext.PrimaryKey
	return i.itrCache.add(&objNext)

}

func (i *Idx64) previousPrimary(iterator int, primary *uint64) int {

	idx, _ := i.context.DB.GetIndex("byPrimary", &entity.SecondaryObjectI64{})

	if iterator < -1 {
		tab := i.itrCache.findTablebyEndIterator(iterator)
		EosAssert(tab != nil, &InvalidTableTterator{}, "not a valid end iterator")

		objTId := entity.SecondaryObjectI64{TId: tab.ID}

		itr, _ := idx.UpperBound(&objTId)
		if idx.CompareIterator(idx.Begin(), idx.End()) || idx.CompareBegin(itr) {
			return -1
		}

		itr.Prev()
		objPrev := entity.SecondaryObjectI64{}
		itr.Data(&objPrev)

		if objPrev.TId != tab.ID {
			return -1
		}

		*primary = objPrev.PrimaryKey
		return i.itrCache.add(&objPrev)
	}

	obj := (i.itrCache.get(iterator)).(*entity.SecondaryObjectI64)
	itr := idx.IteratorTo(obj)

	if idx.CompareBegin(itr) {
		return -1
	}

	itr.Prev()
	objNext := entity.SecondaryObjectI64{}
	itr.Data(&objNext)

	if objNext.TId != obj.TId {
		return -1
	}
	*primary = objNext.PrimaryKey
	return i.itrCache.add(&objNext)
}

func (i *Idx64) get(iterator int, secondary *uint64, primary *uint64) {
	obj := (i.itrCache.get(iterator)).(*entity.SecondaryObjectI64)

	*primary = obj.PrimaryKey
	*secondary = obj.SecondaryKey
}
