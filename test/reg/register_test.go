package reg

import (
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"log"
	"testing"
)

func Test_register(t *testing.T) {
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, "software\\lollipop", registry.ALL_ACCESS)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Close()

	//subKey, _, err := registry.CreateKey(key, "test", registry.ALL_ACCESS)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer subKey.Close()

	key1, err := registry.OpenKey(registry.LOCAL_MACHINE, "software\\lollipop", registry.ALL_ACCESS)
	if err != nil {
		log.Fatal(err)
	}
	defer key1.Close()

	//err = key.SetStringValue("K", "V")
	//if err != nil {
	//	log.Fatal(err)
	//}

	v, _, err := key.GetStringValue("config")
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(v, vt)

	data := make([]map[string]interface{}, 0)
	e := json.Unmarshal([]byte(v), &data)
	if e != nil {
		panic(e)
	}
	for _, person := range data {
		if person["unionid"] == "o4dYU1gcUceNS6WzFeydGG3jgjto" {
			person["isLogined"] = false
		}
	}

	out, er := json.MarshalIndent(data, "", "\t")
	if er != nil {
		panic(er)
	}
	fmt.Print(string(out))

	err = key.SetStringValue("config", string(out))
	if err != nil {
		log.Fatal(err)
	}

	/*
		kns, err := key.ReadSubKeyNames(0)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(kns)
	*/
	/*
		vns, err := key.ReadValueNames(0)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(vns)
	*/
	/*
		err = key.DeleteValue("A")
		if err != nil {
			log.Fatal(err)
		}
	*/

	/*
		err = registry.DeleteKey(registry.LOCAL_MACHINE, "test")
		if err != nil {
			log.Fatal(err)
		}
	*/
	/*
		err = registry.DeleteKey(key, "test_sub")
		if err != nil {
			log.Fatal(err)
		}
	*/
}
