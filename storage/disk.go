package storage

import (
	"io/ioutil"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

type Disk struct {
	data     map[string]interface{}
	filename string
	mutex    sync.RWMutex
}

func NewDisk(filename string) (*Disk, error) {
	d := &Disk{
		data:     make(map[string]interface{}),
		filename: filename,
	}

	err := d.load()
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Disk) load() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	file, err := ioutil.ReadFile(d.filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(file, &d.data)
	if err != nil {
		return err
	}

	return nil
}

func (d *Disk) Get(key string) (interface{}, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	value, ok := d.data[key]
	return value, ok
}

func (d *Disk) Set(key string, value interface{}) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.data[key] = value
	err := d.flush()

	return err
}

func (d *Disk) flush() error {
	yamlData, err := yaml.Marshal(&d.data)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(d.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(yamlData)
	return err
}
