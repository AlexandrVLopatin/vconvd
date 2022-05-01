package manager

import (
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"

	"vconvd/model"
)

type DataStorage struct {
	DbFile string
	_db    *bolt.DB
}

func (d *DataStorage) db() (*bolt.DB, error) {
	if d._db != nil {
		return d._db, nil
	}

	return nil, fmt.Errorf("Db is not open")
}

func (d *DataStorage) Open() error {
	db, err := bolt.Open(d.DbFile, 0600, nil)
	if err != nil {
		return err
	}

	d._db = db
	return nil
}

func (d *DataStorage) Close() {
	d._db.Close()
}

func (d *DataStorage) CreateNewDb() error {
	err := d.Open()
	if err != nil {
		return err
	}

	defer d._db.Close()

	return d._db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("task"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}

func (d *DataStorage) CreateTask(task *model.ConversionTask) error {
	db, err := d.db()
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("task"))

		buf, err := json.Marshal(task)
		if err != nil {
			return err
		}

		return b.Put([]byte(task.ID), buf)
	})
}

func (d *DataStorage) DeleteTask(task *model.ConversionTask) error {
	db, err := d.db()
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("task"))
		return b.Delete([]byte(task.ID))
	})
}
