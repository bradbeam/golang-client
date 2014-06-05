package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"git.openstack.org/stackforge/golang-client.git/identity"
	"git.openstack.org/stackforge/golang-client.git/objectstorage"
	"io/ioutil"
	"time"
)

func main() {
	config := getConfig()

	// Before working with object storage we need to authenticate with a project
	// that has active object storage.
	auth, err := identity.AuthUserNameTenantName(config.Host,
		config.Username,
		config.Password,
		config.ProjectName)
	if err != nil {
		panicString := fmt.Sprint("There was an error authenticating:", err)
		panic(panicString)
	}
	if !auth.Access.Token.Expires.After(time.Now()) {
		panic("There was an error. The auth token has an invalid expiration.")
	}

	// Find the endpoint for object storage.
	url := ""
	for _, svc := range auth.Access.ServiceCatalog {
		if svc.Type == "object-store" {
			url = svc.Endpoints[0].PublicURL + "/"
			break
		}
	}
	if url == "" {
		panic("object-store url not found during authentication")
	}

	hdr, err := objectstorage.GetAccountMeta(url, auth.Access.Token.Id)
	if err != nil {
		panicString := fmt.Sprint("There was an error getting account metadata:", err)
		panic(panicString)
	}

	// Create a new container.
	if err = objectstorage.PutContainer(url+config.Container, auth.Access.Token.Id,
		"X-Log-Retention", "true"); err != nil {
		panicString := fmt.Sprint("PutContainer Error:", err)
		panic(panicString)
	}

	// Get a list of all the containers at the selected endoint.
	containersJson, err := objectstorage.ListContainers(0, "", url, auth.Access.Token.Id)
	if err != nil {
		panic(err)
	}

	type containerType struct {
		Name         string
		Bytes, Count int
	}
	containersList := []containerType{}

	if err = json.Unmarshal(containersJson, &containersList); err != nil {
		panic(err)
	}

	found := false
	for i := 0; i < len(containersList); i++ {
		if containersList[i].Name == config.Container {
			found = true
		}
	}
	if !found {
		panic("Created container is missing from downloaded containersList")
	}

	// Set and Get container metadata.
	if err = objectstorage.SetContainerMeta(url+config.Container, auth.Access.Token.Id,
		"X-Container-Meta-fubar", "false"); err != nil {
		panic(err)
	}

	hdr, err = objectstorage.GetContainerMeta(url+config.Container, auth.Access.Token.Id)
	if err != nil {
		panicString := fmt.Sprint("GetContainerMeta Error:", err)
		panic(panicString)
	}
	if hdr.Get("X-Container-Meta-fubar") != "false" {
		panic("container meta does not match")
	}

	// Create an object in a container.
	var fContent []byte
	srcFile := "10-objectstore.go"
	fContent, err = ioutil.ReadFile(srcFile)
	if err != nil {
		panic(err)
	}

	object := config.Container + "/" + srcFile
	if err = objectstorage.PutObject(&fContent, url+object, auth.Access.Token.Id,
		"X-Object-Meta-fubar", "false"); err != nil {
		panic(err)
	}
	objectsJson, err := objectstorage.ListObjects(0, "", "", "", "",
		url+config.Container, auth.Access.Token.Id)

	type objectType struct {
		Name, Hash, Content_type, Last_modified string
		Bytes                                   int
	}
	objectsList := []objectType{}

	if err = json.Unmarshal(objectsJson, &objectsList); err != nil {
		panic(err)
	}
	found = false
	for i := 0; i < len(objectsList); i++ {
		if objectsList[i].Name == srcFile {
			found = true
		}
	}
	if !found {
		panic("created object is missing from the objectsList")
	}

	// Manage object metadata
	if err = objectstorage.SetObjectMeta(url+object, auth.Access.Token.Id,
		"X-Object-Meta-fubar", "true"); err != nil {
		panicString := fmt.Sprint("SetObjectMeta Error:", err)
		panic(panicString)
	}
	hdr, err = objectstorage.GetObjectMeta(url+object, auth.Access.Token.Id)
	if err != nil {
		panicString := fmt.Sprint("GetObjectMeta Error:", err)
		panic(panicString)
	}
	if hdr.Get("X-Object-Meta-fubar") != "true" {
		panicString := fmt.Sprint("SetObjectMeta Error:", err)
		panic(panicString)
	}

	// Retrieve an object and check that it is the same as what as uploaded.
	_, body, err := objectstorage.GetObject(url+object, auth.Access.Token.Id)
	if err != nil {
		panicString := fmt.Sprint("GetObject Error:", err)
		panic(panicString)
	}
	if !bytes.Equal(fContent, body) {
		panicString := fmt.Sprint("GetObject Error:", "byte comparison of uploaded != downloaded")
		panic(panicString)
	}

	// Duplication (Copy) an existing object.
	if err = objectstorage.CopyObject(url+object, "/"+object+".dup", auth.Access.Token.Id); err != nil {
		panicString := fmt.Sprint("CopyObject Error:", err)
		panic(panicString)
	}

	// Delete the objects.
	if err = objectstorage.DeleteObject(url+object, auth.Access.Token.Id); err != nil {
		panicString := fmt.Sprint("DeleteObject Error:", err)
		panic(panicString)
	}
	if err = objectstorage.DeleteObject(url+object+".dup", auth.Access.Token.Id); err != nil {
		panicString := fmt.Sprint("DeleteObject Error:", err)
		panic(panicString)
	}

	// Delete the container that was previously created.
	if err = objectstorage.DeleteContainer(url+config.Container,
		auth.Access.Token.Id); err != nil {
		panicString := fmt.Sprint("DeleteContainer Error:", err)
		panic(panicString)
	}
}