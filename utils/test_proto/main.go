package main

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
)

func main() {
	article := golosovaniepb.Article{
		Title:       "The world needs change 🌳",
		Description: "",
		Created:     1596806111080,
		Updated:     0,
		Public:      true,
		Promoted:    false,
		Type:        golosovaniepb.Type_NEWS,
		Review:      golosovaniepb.Review_UNSPECIFIED2,
		Comments:    []string{"Nice one", "Thank you"},
		Backlinks:   []string{},
	}
	out, err := proto.Marshal(&article)
	if err != nil {
		panic(err)
	}
	referenceSerialization := "0a1b54686520776f726c64206e65656473206368616e676520f09f8cb318e8bebec8bc2e280138024a084e696365206f6e654a095468616e6b20796f75"
	if hex.EncodeToString(out) == referenceSerialization {
		fmt.Println("SUCCESS, serialization is deterministic")
	} else {
		fmt.Println("ERROR, serialization differs from reference")
	}
}
