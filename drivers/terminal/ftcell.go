package term

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"fmt"
	"strings"
	"github.com/atdiar/particleui"
)

var allowdatapersistence = ui.NewConstructorOption("datapersistence", func(e *ui.Element) *ui.Element {
	d:= getDocumentRef(e)

	e.WatchEvent("datastore-load",e,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{		
		LoadFromStorage(evt.Origin())
		return false
	}))

	d.WatchEvent("document-loaded",d,ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		e.TriggerEvent("datastore-load")
		return false
	}).RunASAP().RunOnce())
	

	d.OnBeforeUnactive(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
		PutInStorage(e)
		return false
	}))
	return e
})

var diskStorage *diskStore

type diskStore struct {
	filePath string
	file     *os.File
}

func initDiskStorage(filePath string) error {
    diskStorage = &diskStore{
        filePath: filePath,
    }

    // Check if file exists
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        // File does not exist, initialize it with "zui-connected"
        if err := diskStorage.ensureOpen(); err != nil {
            return err
        }

        if err := diskStorage.Set("zui-connected", true); err != nil {
            return fmt.Errorf("failed to set default key: %v", err)
        }
    } else {
        // File exists, check for the "zui-connected" key to ensure it's not tampered with
        if err := diskStorage.ensureOpen(); err != nil {
            return err
        }

        val, err := diskStorage.Get("zui-connected")
        if err != nil || val == nil || val.(bool) != true {
            return fmt.Errorf("the data file seems to have been tampered with or is not initialized correctly")
        }
    }

    return nil
}


func (s *diskStore) ensureOpen() error {
	if s.file == nil {
		var err error
		s.file, err = os.OpenFile(s.filePath, os.O_RDWR|os.O_CREATE, 0644)
		return err
	}
	return nil
}


func (s *diskStore) Close() error {
	if s.file != nil {
		err := s.file.Close()
		s.file = nil 
		return err
	}
	return nil
}

func (s *diskStore) Get(key string) (any, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(s.file)
	if err != nil {
		return nil, err
	}

	var storageMap map[string]any
	err = json.Unmarshal(data, &storageMap)
	if err != nil {
		return nil, err
	}

	value, exists := storageMap[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found", key)
	}
	return value, nil
}

func (s *diskStore) Set(key string, value any) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}

	data, err := io.ReadAll(s.file)
	if err != nil {
		return err
	}

	var storageMap map[string]any
	if err := json.Unmarshal(data, &storageMap); err != nil {
		storageMap = make(map[string]any)
	}

	storageMap[key] = value
	return s.writeToFile(storageMap)
}

func (s *diskStore) Delete(key string) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}

	data, err := io.ReadAll(s.file)
	if err != nil {
		return err
	}

	var storageMap map[string]any
	if err := json.Unmarshal(data, &storageMap); err != nil {
		return err
	}

	delete(storageMap, key)
	return s.writeToFile(storageMap)
}

// clear erases the file content.
func (s *diskStore) clear() error {
	s.file.Seek(0, 0)
	return s.file.Truncate(0)
}

// writeToFile is a helper to marshal the map and write it to the file.
func (s *diskStore) writeToFile(storageMap map[string]any) error {
	newData, err := json.Marshal(storageMap)
	if err != nil {
		return err
	}

	s.clear()
	_, err = s.file.Write(newData)

	return err
}


// Let's add disk storage for Element properties.
func storer(s string) func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
	return func(element *ui.Element, category string, propname string, value ui.Value, flags ...bool) {
		if category != "data"{
			return 
		}
		store := diskStorage
		_,err:= store.Get("zui-connected")
		if err != nil{
			log.Print("storage is disconnected")
			return
		}

		props := make([]any, 0, 64)

		c,ok:= element.Properties.Categories[category]
		if !ok{
			props = append(props, propname)
			store.Set(element.ID, props) 
		} else{
			for k:= range c.Local{
				props = append(props, k)
			}
			store.Set(element.ID, props)
		}
	
		item := value.RawValue()
		v := stringify(item)
		store.Set(strings.Join([]string{element.ID, category, propname}, "/"),js.ValueOf(v))
		return
	}
}


var store = storer("disk")

func loader(s string) func(e *ui.Element) error { // abstractjs
	return func(e *ui.Element) error {
		
		store := diskStorage
		_,err:= store.Get("zui-connected")
		if err!= nil{
			return errors.New("storage is disconnected")
		}
		id := e.ID

		// Let's retrieve the category index for this element, if it exists in the sessionstore
		jsonprops, err := store.Get(id)
		if jsonprops==nil { // TODO REVIEW THIS AS WE CHANGED FROM A BOOL TO AN ERROR
			return nil // Not necessarily an error in the general case. element just does not exist in store
		}

		properties := make([]string, 0, 64)
		err = json.Unmarshal([]byte(jsonprops.String()), &properties)
		if err != nil {
			return err
		}

		category:= "data"
		uiloaders:= make([]func(),0,64)

		for _, property := range properties {
			// let's retrieve the propname (it is suffixed by the proptype)
			// then we can retrieve the value
			// log.Print("debug...", category, property) // DEBUG

			propname := property
			jsonvalue, ok := store.Get(strings.Join([]string{e.ID, category, propname}, "/"))
			if ok {					
				var rawvaluemapstring string
				err = json.Unmarshal([]byte(jsonvalue.String()), &rawvaluemapstring)
				if err != nil {
					return err
				}
				
				rawvalue := make(map[string]any)
				err = json.Unmarshal([]byte(rawvaluemapstring), &rawvalue)
				if err != nil {
					return err
				}
				val:= ui.ValueFrom(rawvalue)

				ui.LoadProperty(e, category, propname, val)
				if category == "data"{
					uiloaders = append(uiloaders, func(){
						if e.IsRenderData(propname){
							e.SetUI(propname, val)
						}
					})
				}
				//log.Print("LOADED PROPMAP: ", e.Properties, category, propname, rawvalue.Value()) // DEBUG
			}
		}

		
		//log.Print(categories, properties) //DEBUG
		
		e.OnRegistered(ui.NewMutationHandler(func(evt ui.MutationEvent)bool{
			for _,load:= range uiloaders{
				load()
			}
			return false
		}).RunOnce())
		
		return nil
	}
}

var load = loader("disk")

func clearer(s string) func(element *ui.Element){ // abstractjs
	return func(element *ui.Element){
		store := jsStore{js.Global().Get(s)}
		_,ok:= store.Get("zui-connected")
		if !ok{
			return 
		}
		id := element.ID
		category:= "data"

		// Let's retrieve the category index for this element, if it exists in the sessionstore
		jsonproperties, ok := store.Get(id)
		if !ok {
			return
		}

		properties := make([]string, 0, 50)
		
		err := json.Unmarshal([]byte(jsonproperties.String()), &properties)
		if err != nil {
			store.Delete(id)
			panic("An error occured when removing an element from storage. It's advised to reinitialize " + s)
		}

		for _, property := range properties {
			// let's retrieve the propname (it is suffixed by the proptype)
			// then we can retrieve the value
			// log.Print("debug...", category, property) // DEBUG

			store.Delete(strings.Join([]string{id, category, property}, "/")) 
		}

		store.Delete(id)
	}
}

var clear = clearer("disk")

// isPersisted checks whether an element exist in storage already
func isPersisted(e *ui.Element) bool{
	pmode:=ui.PersistenceMode(e)

	var s string
	switch pmode{
	case"disk":
		s = "disk"
	default:
		return false
	}

	store := jsStore{js.Global().Get(s)}
	_, ok := store.Get(e.ID)
	return ok
}

func stringify(v interface{}) string {
	res, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(res)
}