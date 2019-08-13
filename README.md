## STF
STF — Sort That File.

Utility which may be used to sort **binary** files on systems with limited RAM.

### Usage
```
➭ ./stf --help

Usage of ./stf:
  -batch int
    	Batch size of one bucket (default 33554432)
  -bytes int
    	Values size in bytes (default 8)
  -file string
    	Input file that should be sorted
  -out string
    	Output sorted file
  -tmpdir string
    	Set tmp dir for temporary files (default "./")
  -unsigned
    	Set if values should be interpreted as unsigned
```

There is an option `-batch` that may be used to limit RAM usage of the app (set in bytes).

`-bytes` option needs to specify what type of values contains in file. Possible options is: 1 (byte, int8 or uint8), 2 (int16 or uint16), 4 (int32 or uint32), or 8 (int64 or uint64)

`-unsigned` options may be used to specify unsigned type of data in file (by default its interpreted as signed)

### Instalation
Simplest way to install is just
```
go get github.com/wierdbytes/stf
```

Also you can download it from releases page

### Examples
```
➭ head -c 10485760 /dev/urandom > 10m.dumb
➭ stf --bytes 2 --unsigned --batch 2097152 --file 10m.dumb
➭ ls -al 10m.dumb.sorted
```

Also you could pipe input into `stf` like so:
```
➭ cat 10m.dumb | stf --bytes 2 --unsigned --batch 2097152
➭ ls -al .sorted
```

### Things to improve
- Refactor the code (now he just works)
- Make use of parallel execution (now it 1 threaded)
- Cover code by tests (I will, I swear)
- Add support of littleEndian
- ...