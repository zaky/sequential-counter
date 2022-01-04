package p

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	store "cloud.google.com/go/storage"
)

const (
	projectID = `my-project`
	Bucket    = `counters`
)

var (
	ctx    context.Context
	client *store.Client
)

func init() {
	ctx = context.Background()

	var err error
	client, err = store.NewClient(ctx)
	if err != nil {
		panic("fail to create storage client " + err.Error())
	}

}

// NewJobFile prints information about a GCS event.
func Deal(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		fmt.Printf("error marshal input: %s %s", Bucket, err)
		fmt.Fprintf(w, "error marshal input: %s %s", Bucket, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Println("input", input.Path)
	//Read number from object.counter
	bkt := client.Bucket(Bucket)
	obj := bkt.Object(input.Path)
	val, err := run(obj)
	if err != nil {
		fmt.Fprint(w, err)
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "%d", val)
}

func run(obj *store.ObjectHandle) (int, error) {
	file, err := obj.NewReader(ctx)

	if err != nil {
		return 0, fmt.Errorf("error opening file: %s/%s %s", Bucket, obj.ObjectName(), err)
	}
	defer file.Close()
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return 0, fmt.Errorf("error reading file: %s/%s %s", Bucket, obj.ObjectName(), err)
	}
	objAttrs, err := obj.Attrs(ctx)
	if err != nil {
		return 0, fmt.Errorf("error reading metadata: %s/%s %s", Bucket, obj.ObjectName(), err)
	}
	//fmt.Println("generation", objAttrs.Generation)
	//Add 1 or any other logic
	if _, err := io.Copy(buf, file); err != nil {
		return 0, fmt.Errorf("error reading string: %s/%s %s", Bucket, obj.ObjectName(), err)
	}
	valStr := strings.Replace(buf.String(), "\n", "", -1)
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("error casting to val: %s/%s val %s %s", Bucket, obj.ObjectName(), valStr, err)
	}
	val++
	writer := obj.If(store.Conditions{GenerationMatch: objAttrs.Generation}).NewWriter(ctx)
	// Write some text to obj. This will either create the object or overwrite whatever is there already.
	if _, err := fmt.Fprintf(writer, "%d", val); err != nil {
		return 0, fmt.Errorf("error writing val: %s/%s %s", Bucket, obj.ObjectName(), err)
	}
	// Close, just like writing a file.
	//Write back the value if if-generation-match
	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("error writing file: %s/%s %s", Bucket, obj.ObjectName(), err)
	}

	//If 412 Precondition Failed. Try again 5 times with increase time, 1,3,7,12,20
	//If still error return error
	return val, nil

}
