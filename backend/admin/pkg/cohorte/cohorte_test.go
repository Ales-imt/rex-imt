package cohorte

import (
	"back-rex-admin/pkg/cohorte/testdata"
	"back-rex-common/pkg/services"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"
)

func TestImportCohorte(t *testing.T) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := addFilePart(writer, "data", testdata.Path("data.json"))
	if err != nil {
		t.Fatalf("Failed to add data file part: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()

	dsn := "host=localhost port=5432 user=postgres password=root dbname=db_rex sslmode=disable"

	pg := services.NewPG(context.Background(), dsn)
	ctx := context.WithValue(req.Context(), services.PgCtxKey2, pg)
	req = req.WithContext(ctx)

	cfg := services.LDAPConfig{
		URL: "ldap://localhost:3890",
		//URL:    "ldap://ldap.mines-ales.fr:389",
		BaseDN: "dc=ema,dc=fr",
	}

	ImportCohorte(rr, req, cfg)

	// Optionally, check output or error rendering
}

func addFilePart(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="`+fieldName+`"; filename="`+file.Name()+`"`)
	partHeader.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}
	return nil
}
