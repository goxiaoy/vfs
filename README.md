### Virtual File System 
Golang virtual file system based on  [afero](https://github.com/spf13/afero)

#### usange
```go

vfs := NewVfs() //vfs implements afero.Fs

vfs.Mount("/", afero.NewMemMapFs()) //second prameter could be any afero.Fs 
vfs.Mount("/abc", afero.NewMemMapFs())
vfs.Mount("/a/b/c/d", afero.NewMemMapFs())

f,err := vfs.Create("/a/test.txt") // Creat file, for all functions see https://github.com/spf13/afero#list-of-all-available-functions
```

#### Blob

Extra blob interface
```go
type Blob interface {
	FS
	Linker
	//TODO
	//Mover
	//Copier
	//Lister
}
```

#### Planned Features

- [ ] Metadata storage
- [ ] Data At Rest Encryption (DARE)


### Thanks to
https://github.com/embeddedgo/go

https://github.com/spf13/afero

https://pkg.go.dev/gocloud.dev/blob

https://github.com/dghubble/trie