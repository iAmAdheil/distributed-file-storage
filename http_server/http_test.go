package http_server

import (
	"encoding/xml"
	"fmt"
	"testing"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/db"
	"github.com/iAmAdheil/distributed-file-storage/db/model"
)

func TestXMLRes(t *testing.T) {
	md := model.Metadata{
		Bucket:      "some-bucket",
		Key:         "some-key",
		Size:        100,
		ContentType: "some content type",
		CreatedAt:   time.Now(),
	}

	lmres := db.ListMetaRes{
		List:      []model.Metadata{md},
		ContToken: "some continuation token",
	}

	res := listhandlerRes{
		Objects:   lmres.List,
		IsTrunc:   true,
		ContToken: lmres.ContToken,
	}

	output, err := xml.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal XML: %v", err)
	}

	// 2. Convert to string for printing/logging
	xmlString := string(output)
	fmt.Printf("Generated XML:\n%s\n", xmlString)
}
