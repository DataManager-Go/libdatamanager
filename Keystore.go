package libdatamanager

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jinzhu/gorm"
)

const (
	// KeystoreDBFile the sqlite DB containing the file-key associations
	KeystoreDBFile = ".keys.db"

	// KeyringService the servicename for the keyring
	KeyringService = "DataManagerCLI-keystore"
)

var (
	// ErrKeyUnavailable if keystore key is unavailable
	ErrKeyUnavailable = errors.New("keyring key is unavailable")
	// ErrKeystoreNoDir error if keystore is no directory
	ErrKeystoreNoDir = errors.New("Keystore is not a directory")
)

// KeystoreFile the keystore row
type KeystoreFile struct {
	gorm.Model
	FileID uint
	Key    string
}

// Keystore a place to store keys
type Keystore struct {
	Path     string
	DB       *gorm.DB
	fileInfo os.FileInfo
}

// NewKeystore create a new keystore
func NewKeystore(path string) *Keystore {
	return &Keystore{
		Path: path,
	}
}

// GetKeystoreFile returns the full path of file
func (store *Keystore) GetKeystoreFile(file string) string {
	return filepath.Join(store.Path, file)
}

// GetKeystoreDataFile returns the keystore db filepath
func (store *Keystore) GetKeystoreDataFile() string {
	return store.GetKeystoreFile(KeystoreDBFile)
}

// Open opens the keystore
func (store *Keystore) Open() error {
	var err error

	// Get Info
	store.fileInfo, err = os.Stat(store.Path)
	if err != nil {
		return err
	}

	if !store.fileInfo.IsDir() {
		return ErrKeystoreNoDir
	}

	// Open DB into memory
	store.DB, err = gorm.Open("sqlite3", store.GetKeystoreDataFile())
	if err != nil {
		return err
	}

	// Migrate DB
	err = store.DB.AutoMigrate(&KeystoreFile{}).Error

	return err
}

// AddKey Inserts key into keystore
func (store *Keystore) AddKey(fileID uint, keyPath string) error {
	_, keyFile := filepath.Split(keyPath)
	return store.DB.Create(&KeystoreFile{
		FileID: fileID,
		Key:    keyFile,
	}).Error
}

// DeleteKey Inserts key into keystore
func (store *Keystore) DeleteKey(fileID uint) (*KeystoreFile, error) {
	file, err := store.GetKeyFile(fileID)
	if err != nil {
		return nil, err
	}
	return file, store.DB.Unscoped().Delete(&file).Error
}

// GetKeyFile returns a keyfile with assigned to the fileID
func (store *Keystore) GetKeyFile(fileID uint) (*KeystoreFile, error) {
	var storeFile KeystoreFile

	// Find in db
	err := store.DB.Model(&KeystoreFile{}).Where("file_id=?", fileID).Find(&storeFile).Error
	if err != nil {
		return nil, err
	}

	return &storeFile, nil
}

// GetKey returns the key assigned to the fileID. If FileID or key was
// not found, error is not nil
func (store *Keystore) GetKey(fileID uint) ([]byte, error) {
	// Get DB filekey
	storefile, err := store.GetKeyFile(fileID)
	if err != nil {
		return nil, err
	}

	// Read keyfile
	return ioutil.ReadFile(store.GetKeystoreFile(storefile.Key))
}

// GetFiles returns a slice containing all keystore Files
func (store *Keystore) GetFiles() ([]KeystoreFile, error) {
	var fileitems []KeystoreFile

	// Find files in DB
	err := store.DB.Find(&fileitems).Error
	if err != nil {
		return nil, err
	}

	return fileitems, nil
}

// GetValidKeyCount reutrns count of valid keys
func (store *Keystore) GetValidKeyCount() (int, error) {
	var validItems int

	// Get files
	fileitems, err := store.GetFiles()
	if err != nil {
		return 0, err
	}

	// Select valid files
	for i := range fileitems {
		_, err := os.Stat(store.GetKeystoreFile(fileitems[i].Key))
		if err == nil {
			validItems++
		}
	}

	return validItems, nil
}

// GetFileInfo returns fileinfo for the keystore
func (store *Keystore) GetFileInfo() *os.FileInfo {
	return &store.fileInfo
}

// Close closes the keystore
func (store *Keystore) Close() {
	if store == nil || store.DB == nil {
		return
	}
	store.DB.Close()
}
