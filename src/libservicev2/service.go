package libservicev2

import (
	"github.com/jinzhu/gorm"
	"container/list"
	"app"
	"errors"
	"util/common"
)

var dbPool *DbConnPool

func SetPool(pool *DbConnPool) {
	dbPool = pool
}

func transformNotFoundErr(err error) error {
	if gorm.IsRecordNotFoundError(err) {
		return nil
	}
	return err
}

// get fileId from table file by md5,
func GetFileIdByMd5(md5 string, dao *DAO) (id int64, e error) {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return 0, ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}
	var fileDO app.FileDO
	e = dao.Query(func(db *gorm.DB) error {
		return transformNotFoundErr(db.Table("file").Select("id").Where("md5 = ?", md5).Scan(&fileDO).Error)
	})
	return fileDO.Id, e
}


// get fileId from table file by md5,
func GetPartIdByMd5(md5 string, dao *DAO) (id int64, e error) {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return 0, ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}
	var partDO app.PartDO
	e = dao.Query(func(db *gorm.DB) error {
		return transformNotFoundErr(db.Table("part").Select("id").Where("md5 = ?", md5).Scan(&partDO).Error)
	})
	return partDO.Id, e
}


// insert new file to table file,
// if file exists, file id will replaced by existing id.
func InsertFile(file *app.FileVO, dao *DAO) error {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}
	return dao.DoTransaction(func(db *gorm.DB) error {
		e1 := db.Where(app.FileDO{Md5: file.Md5}).FirstOrCreate(file).Error
		if e1 != nil && transformNotFoundErr(e1) != nil {
			return e1
		}
		for i := range file.Parts {
			e2 := insertFilePart(&file.Parts[i], db)
			if e2 != nil {
				return e2
			}
			relation := &app.FilePartRelationDO{FileId: file.Id, PartId: file.Parts[i].Id}
			e3 := insertFilePartRelation(relation, db)
			if e3 != nil {
				return e3
			}
		}
		return nil
	})
}

// insert new file to table part,
// if part exists, part id will replaced by existing id.
func insertFilePart(part *app.PartDO, tx *gorm.DB) error {
	return transformNotFoundErr(tx.Where(app.PartDO{Md5: part.Md5}).FirstOrCreate(part).Error)
}

// insert new file_part relation to table relation_file_part,
// if relation exists, relation id will replaced by existing id.
func insertFilePartRelation(relation *app.FilePartRelationDO, tx *gorm.DB) error {
	return transformNotFoundErr(tx.Where(app.FilePartRelationDO{FileId: relation.FileId, PartId: relation.PartId}).FirstOrCreate(relation).Error)
}

// save current app uuid to table sys,
// if uuid already exists, skip
func ConfirmAppUUID(uuid string) (ret string, e error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return "", ef
	}
	defer dbPool.ReturnDB(dao)
	uuidDO := &app.SysDO{Key: "uuid", Value: uuid}
	return uuidDO.Value, dao.Query(func(db *gorm.DB) error {
		return transformNotFoundErr(db.Where(app.SysDO{Key: "uuid"}).FirstOrCreate(uuidDO).Error)
	})
}


// update table tracker
func UpdateTrackerInfo(tracker *app.TrackerDO) error {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return ef
	}
	defer dbPool.ReturnDB(dao)
	return dao.DoTransaction(func(db *gorm.DB) error {
		return transformNotFoundErr(db.Save(tracker).Error)
	})
}

// update table tracker
func GetTrackerInfo(uuid string) (*app.TrackerDO, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var ret app.TrackerDO
	var rowAffect int64
	e := dao.Query(func(db *gorm.DB) error {
		result := db.Table("tracker").Where("uuid = ?", uuid).First(&ret)
		rowAffect = result.RowsAffected
		return transformNotFoundErr(result.Error)
	})
	if rowAffect == 0 {
		return nil, e
	}
	return &ret, e
}


// get ready push files to tracker.
func GetReadyPushFiles(trackerUUID string) (*list.List, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var files = list.New()
	
	return files, dao.Query(func(db *gorm.DB) error {
		total := &app.Total{}
		result := db.Raw("select count(*) as count from file f where f.id > (select local_push_id from tracker a where a.uuid = ?)", trackerUUID).Scan(total)
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		// limit result set size, not too big, not too small.
		limit := total.Count / 10
		if limit > 50 {
			limit = 50
		}
		if limit < 20 {
			limit = 20
		}

		rows, e := db.Raw("select * from file f where f.id > (select local_push_id from tracker a where a.uuid = ?) limit ?", trackerUUID, limit).Rows()
		if transformNotFoundErr(e) != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			file := &app.FileVO{}
			e1 := db.ScanRows(rows, file)
			if e1 != nil {
				return e1
			}
			files.PushBack(file)
		}
		fileIds := make([]int64, files.Len())
		index := 0
		for ele := files.Front(); ele != nil; ele = ele.Next() {
			fileIds[index] = ele.Value.(*app.FileVO).Id
			index++
		}
		rows2, e2 := db.Raw("select a.*, r.fid from relation_file_part r left join part a " +
			"on r.pid = a.id where r.fid in(?)", fileIds).Rows()
		if transformNotFoundErr(e2) != nil {
			return e2
		}
		defer rows2.Close()
		var parts = list.New()
		for rows2.Next() {
			part := &app.PartVO{}
			e1 := db.ScanRows(rows2, part)
			if e1 != nil {
				return e1
			}
			parts.PushBack(part)
		}
		for fileEle := files.Front(); fileEle != nil; fileEle = fileEle.Next() {
			file := fileEle.Value.(*app.FileVO)
			ls := list.New()
			for partEle := parts.Front(); partEle != nil; partEle = partEle.Next() {
				part := partEle.Value.(*app.PartVO)
				if part.FileId == file.Id {
					ls.PushBack(part)
				}
			}
			file.SetPartsFromVO(ls)
		}
		return nil
	})
}


// get full file by file md5
// finish:
// 0|1 : file download finish flag
// 2   : not add 'finish' query parameter
func GetFullFileByMd5(md5 string, finish int) (*app.FileVO, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var addOn = ""
	if finish == 1 {
		addOn = " and finish = 1"
	} else if finish == 0 {
		addOn = " and finish = 0"
	}

	var file app.FileVO
	var rowAffect int64 = 0
	err := dao.Query(func(db *gorm.DB) error {
		result := db.Model(&file).Where("md5 = ? " + addOn, md5).First(&file)
		rowAffect = result.RowsAffected
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		if rowAffect == 0 {
			return nil
		}
		rows, e2 := db.Raw("select a.* from relation_file_part r left join part a " +
			"on r.pid = a.id where r.fid = ?", file.Id).Rows()
		if transformNotFoundErr(e2) != nil {
			return e2
		}
		defer rows.Close()
		var parts = list.New()
		for rows.Next() {
			part := &app.PartDO{}
			e1 := db.ScanRows(rows, part)
			if e1 != nil {
				return e1
			}
			parts.PushBack(part)
		}
		file.SetParts(parts)
		return nil
	})
	if rowAffect == 0 {
		return nil, err
	}
	return &file, err
}

// get full file by file md5
// finish:
// 0|1 : file download finish flag
// 2   : not add 'finish' query parameter
func GetFullFileById(fid int64, finish int) (*app.FileVO, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var addOn = ""
	if finish == 1 {
		addOn = " and finish = 1"
	} else if finish == 0 {
		addOn = " and finish = 0"
	}

	var file app.FileVO
	var rowAffect int64 = 0
	err := dao.Query(func(db *gorm.DB) error {
		result := db.Model(&file).Where("id = ? " + addOn, fid).First(&file)
		rowAffect = result.RowsAffected
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		if rowAffect == 0 {
			return nil
		}
		rows, e2 := db.Raw("select a.* from relation_file_part r left join part a " +
			"on r.pid = a.id where r.fid = ?", file.Id).Rows()
		if transformNotFoundErr(e2) != nil {
			return e2
		}
		defer rows.Close()
		var parts = list.New()
		for rows.Next() {
			part := &app.PartDO{}
			e1 := db.ScanRows(rows, part)
			if e1 != nil {
				return e1
			}
			parts.PushBack(part)
		}
		file.SetParts(parts)
		return nil
	})
	if rowAffect == 0 {
		return nil, err
	}
	return &file, err
}


// update file finish status
// status: 0|1
func UpdateFileFinishStatus(id int64, status int, dao *DAO) error {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}

	return dao.DoTransaction(func(db *gorm.DB) error {
		result := db.Table("file").Where("id = ?", id).Update("finish", status)
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		return nil
	})
}


// get files start from specify id,
// onlymine: used by storage when push file.
func GetFullFilesFromId(id int64, onlymine bool, group string, limit int) (*list.List, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	params := make([]interface{}, 4)
	params[0] = id
	params[1] = group
	var query = "select * from file where id > ? and grop = ?"
	if onlymine {
		query += " and instance = ? limit ?"
		params[2] = app.INSTANCE_ID
		params[3] = limit
	} else {
		query += " and 'a'=? limit ?"
		params[2] = "a"
		params[3] = limit
	}

	var files = list.New()
	err := dao.Query(func(db *gorm.DB) error {
		rows, e := db.Raw(query, params...).Rows()
		if transformNotFoundErr(e) != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			file := &app.FileVO{}
			e1 := db.ScanRows(rows, file)
			if e1 != nil {
				return e1
			}
			files.PushBack(file)
		}
		if files.Len() == 0 {
			return nil
		}
		fileIds := make([]int64, files.Len())
		index := 0
		for ele := files.Front(); ele != nil; ele = ele.Next() {
			fileIds[index] = ele.Value.(*app.FileVO).Id
			index++
		}
		rows2, e2 := db.Raw("select a.*, r.fid from relation_file_part r left join part a " +
			"on r.pid = a.id where r.fid in(?)", fileIds).Rows()
		if transformNotFoundErr(e2) != nil {
			return e2
		}
		defer rows2.Close()
		var parts = list.New()
		for rows2.Next() {
			part := &app.PartVO{}
			e1 := db.ScanRows(rows2, part)
			if e1 != nil {
				return e1
			}
			parts.PushBack(part)
		}
		for fileEle := files.Front(); fileEle != nil; fileEle = fileEle.Next() {
			file := fileEle.Value.(*app.FileVO)
			ls := list.New()
			for partEle := parts.Front(); partEle != nil; partEle = partEle.Next() {
				part := partEle.Value.(*app.PartVO)
				if part.FileId == file.Id {
					ls.PushBack(part)
				}
			}
			file.SetPartsFromVO(ls)
		}
		return nil

	})
	return files, err
}


// get storage info by uuid
func GetStorageByUUID(uuid string) (*app.StorageDO, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var storageDO app.StorageDO
	var rowAffect int64
	err := dao.Query(func(db *gorm.DB) error {
		result := db.Table("storage").Where("uuid", uuid).First(&storageDO)
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		rowAffect = result.RowsAffected
		return nil
	})
	if rowAffect == 0 {
		return nil, err
	}
	return &storageDO, err
}


func ExistsStorage(uuid string) (bool, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return false, ef
	}
	defer dbPool.ReturnDB(dao)

	total := &app.Total{}
	total.Count = 0
	err := dao.Query(func(db *gorm.DB) error {
		result := db.Raw("select count(*) as count from storage f where f.uuid = ?", uuid).Scan(total)
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return common.ConvertBoolFromInt(total.Count), err
}


// save or update storage info
func SaveStorage(storage *app.StorageDO) error {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return ef
	}
	defer dbPool.ReturnDB(dao)

	return dao.DoTransaction(func(db *gorm.DB) error {
		result := db.Model(storage).Save(storage)
		if result.RowsAffected == 0 {
			return errors.New("error insert storage: row affect is 0")
		}
		return nil
	})
}


// query system statistic:
// - file count
// - finish file count
// - total group disk space (include placeholder)
func QuerySystemStatistic() (*app.Statistic, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var statistic app.Statistic
	err := dao.Query(func(db *gorm.DB) error {
		result := db.Raw(`select * from (
						(select count(*) files from file a),
						(select count(*) finish from file a where a.finish = 1),
						(select case when sum(b.size) is null then 0 else sum(b.size) end disk from part b)  )`).Scan(&statistic)
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		return nil
	})
	return &statistic, err
}


// get all web trackers which is not deleted.
func GetAllWebTrackers() (*list.List, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var trackers = list.New()
	err := dao.Query(func(db *gorm.DB) error {
		rows, e := db.Table("web_tracker").Where("status != ?", app.STATUS_DELETED).Rows()
		if transformNotFoundErr(e) != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			webTracker := &app.WebTrackerDO{}
			e1 := db.ScanRows(rows, webTracker)
			if e1 != nil {
				return e1
			}
			trackers.PushBack(webTracker)
		}
		return nil
	})
	return trackers, err
}


// insert web tracker
func InsertWebTracker(webTracker *app.WebTrackerDO, dao *DAO) error {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}
	return dao.DoTransaction(func(db *gorm.DB) error {
		result := db.Table("web_tracker").Save(webTracker)
		if result.RowsAffected == 0 {
			return errors.New("error insert web tracker")
		}
		return result.Error
	})
}


// insert web tracker
func UpdateWebTrackerStatus(trackerUuid string, status int, dao *DAO) error {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}
	return dao.DoTransaction(func(db *gorm.DB) error {
		result := db.Table("web_tracker").Where("uuid = ?", trackerUuid).Update("status", status)
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		return nil
	})
}


// insert web storage and relation with web tracker
func InsertWebStorage(trackerUuid string, webStorage *app.WebStorageDO, dao *DAO) error {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}
	return dao.DoTransaction(func(db *gorm.DB) error {
		result := db.Table("web_storage").Save(webStorage)
		if result.Error != nil {
			return result.Error
		}
		relation :=&app.RelationWebTrackerStorageDO{
			TrackerUuid: trackerUuid,
			StorageUuid: webStorage.Uuid,
		}
		result1 := db.Table("relation_web_tracker_storage").Save(relation)
		if result1.Error != nil {
			return result1.Error
		}
		return nil
	})
}


// insert into table web_storage_log
func InsertWebStorageLog(webStorage *app.WebStorageLogDO, dao *DAO) error {
	if dao == nil {
		dao1, ef := dbPool.GetDB()
		if ef != nil {
			return ef
		}
		dao = dao1
		defer dbPool.ReturnDB(dao)
	}
	webStorage.Id = 0
	return dao.DoTransaction(func(db *gorm.DB) error {
		return db.Table("web_storage_log").Create(webStorage).Error
	})
}



// method is used by tracker for statistic
func GetFileCount() int {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return 0
	}
	defer dbPool.ReturnDB(dao)

	total := &app.Total{}
	total.Count = 0
	dao.Query(func(db *gorm.DB) error {
		db.Raw("select count(*) from file").Scan(total)
		return nil
	})
	return total.Count
}


// used by dashboard
func GetIndexStatistic() (*app.DashboardIndexStatistic, error) {
	dao, ef := dbPool.GetDB()
	if ef != nil {
		return nil, ef
	}
	defer dbPool.ReturnDB(dao)

	var statistic app.DashboardIndexStatistic
	err := dao.Query(func(db *gorm.DB) error {
		result := db.Raw(`select
                (select count(*) from web_tracker where status=1)  as trackers,
                (select count(*) from web_storage where status=1)  as storages,
                (select sum(files) from web_tracker where status=1)  as files,
                (select sum(ioin) from web_storage where status=1)  as ioin,
                (select sum(ioout) from web_storage where status=1)  as ioout,
                (select sum(downloads) from web_storage where status=1)  as downloads,
                (select sum(uploads) from web_storage where status=1)  as uploads`).Scan(&statistic)
		if transformNotFoundErr(result.Error) != nil {
			return result.Error
		}
		return nil
	})
	return &statistic, err
}


