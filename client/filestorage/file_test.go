package filestorage

import (
	"fmt"
	"ipfs-share/client"
	"testing"
)

func TestFile_Share(t *testing.T) {
	network := client.Network{"http://0.0.0.0:6000"}
	f := File{"path", "hash", "owner", []string{"hali", "gali"}, []string{}}
	err := f.Share([]string{"gali", "alma"}, "./data/public/for/", &network)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(f.SharedWith)
}