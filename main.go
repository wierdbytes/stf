package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

var (
	numBytes   int
	bucketSize int
	unsigned   bool
	tmpDir     string
	inFile     string
	outFile    string

	files     []string
	fHandlers []*os.File
)

type bucket struct { // TODO: add littleEndian
	Data        []byte
	Size        int
	Signed      bool
	BucketSize  int
	c           int
	fileCounter int
	tmpPath     string
}

// Len is the number of elements in the collection.
func (b *bucket) Len() int {
	return b.c / b.Size
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (b *bucket) Less(i, j int) bool {
	if b.Signed {
		return b.getSignedElement(i) <= b.getSignedElement(j)
	}
	return b.getUnsignedElement(i) <= b.getUnsignedElement(j)
}

func (b *bucket) getSignedElement(i int) int {
	inSlice := i * b.Size
	element := b.Data[inSlice : inSlice+b.Size]
	switch b.Size {
	case 8:
		return int(binary.BigEndian.Uint64(element))
	case 4:
		return int(binary.BigEndian.Uint32(element))
	case 2:
		return int(binary.BigEndian.Uint16(element))
	case 1:
		return int(element[0])
	default:
		panic(fmt.Sprintf("What the bucket size '%d'?", b.Size))
	}
}

func (b *bucket) getUnsignedElement(i int) uint {
	inSlice := i * b.Size
	element := b.Data[inSlice : inSlice+b.Size]
	switch b.Size {
	case 8:
		return uint(binary.BigEndian.Uint64(element))
	case 4:
		return uint(binary.BigEndian.Uint32(element))
	case 2:
		return uint(binary.BigEndian.Uint16(element))
	case 1:
		return uint(element[0])
	default:
		panic(fmt.Sprintf("What the bucket size '%d'?", b.Size))
	}
}

// Swap swaps the elements with indexes i and j.
func (b *bucket) Swap(i, j int) {
	iIndex := i * b.Size
	jIndex := j * b.Size
	for cFor := 0; cFor < b.Size; cFor++ {
		b.Data[iIndex+cFor], b.Data[jIndex+cFor] = b.Data[jIndex+cFor], b.Data[iIndex+cFor]
	}
}

// Minimum returns index of element with minimum value
func (b *bucket) Minimum() int {
	var ret int
	for i := 0; i < b.c; i++ {
		if b.Less(i, ret) {
			ret = i
		}
	}
	return ret
}

func (b *bucket) RemoveElement(i int) {

}

func (b *bucket) Element(i int) []byte {
	inSlice := i * b.Size
	return b.Data[inSlice : inSlice+b.Size]
}

func (b *bucket) Dump() error {
	filename := filepath.Join(b.tmpPath, fmt.Sprintf("stf.tmp.%d", b.fileCounter))
	errWrite := ioutil.WriteFile(filename, b.Data[:b.c], 0644)
	if errWrite != nil {
		return errWrite
	}
	files = append(files, filename)
	b.fileCounter++
	b.c = 0
	b.Data = make([]byte, bucketSize)
	return nil
}

func main() {
	flag.IntVar(&numBytes, "bytes", 8, "Values size in bytes")
	flag.IntVar(&bucketSize, "batch", 1024*1024*32, "Batch size of one bucket")
	flag.BoolVar(&unsigned, "unsigned", false, "Set if values should be interpreted as unsigned")
	flag.StringVar(&tmpDir, "tmpdir", "./", "Set tmp dir for temporary files")
	flag.StringVar(&inFile, "file", "", "Input file that should be sorted")
	flag.StringVar(&outFile, "out", "", "Output sorted file")
	flag.Parse()

	if numBytes != 8 && numBytes != 4 && numBytes != 2 && numBytes != 1 { // TODO: too ugly
		fmt.Printf("ERROR: --bytes should equals 1, 2, 4 or 8, got '%d'\n", numBytes)
		os.Exit(1)
	}
	if bucketSize%numBytes != 0 {
		fmt.Printf("ERROR: batch must be divided by the byts without remainder\n")
	}

	if outFile == "" {
		outFile = inFile + ".sorted"
	}

	scanner := bufio.NewScanner(os.Stdin)
	if inFile != "" {
		fileHandle, errOpen := os.Open(inFile)
		if errOpen != nil {
			fmt.Printf("ERROR: cant open input file: %s", errOpen)
		}
		defer fileHandle.Close()
		scanner = bufio.NewScanner(fileHandle)
	}

	buck := &bucket{
		Data:       make([]byte, bucketSize),
		Size:       numBytes,
		Signed:     !unsigned, // TODO: fix inversion
		BucketSize: bucketSize,
		tmpPath:    tmpDir,
	}
	scanner.Split(getSplitter(numBytes))
	for scanner.Scan() {
		inBytes := scanner.Bytes()
		copy(buck.Data[buck.c:], inBytes)
		buck.c += len(inBytes)
		if buck.c >= buck.BucketSize {
			sort.Sort(buck)
			errDump := buck.Dump()
			if errDump != nil {
				fmt.Printf("ERROR: %s\n", errDump)
				os.Exit(1)
			}
		}
	}
	if buck.c > 0 {
		sort.Sort(buck)
		errDump := buck.Dump()
		if errDump != nil {
			fmt.Printf("ERROR: %s\n", errDump)
			os.Exit(1)
		}
	}
	scanners := make(map[int]*bufio.Scanner)
	finalBucket := &bucket{
		Data:       make([]byte, len(files)*numBytes),
		Size:       numBytes,
		Signed:     !unsigned, // TODO: fix inversion
		BucketSize: len(files) * numBytes,
		c:          len(files),
	}
	fHandlers = make([]*os.File, len(files))
	for i, filename := range files {
		fHandlers[i], _ = os.Open(filename)
		newScanner := bufio.NewScanner(fHandlers[i])
		newScanner.Split(getSplitter(numBytes))
		if newScanner.Scan() {
			scanners[i] = newScanner
			element := i * finalBucket.Size
			copy(finalBucket.Data[element:element+finalBucket.Size], scanners[i].Bytes())
		}
	}

	os.Remove(outFile)
	outHandler, errOut := os.OpenFile(outFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if errOut != nil {
		fmt.Printf("ERROR: cant open out file: %s", errOut)
	}
	defer outHandler.Close()

	for {
		index := finalBucket.Minimum()
		min := finalBucket.Element(index)
		if _, errWrite := outHandler.Write(min); errWrite != nil {
			fmt.Printf("ERROR: cant write to out file: %s", errWrite)
		}
		if scanners[index].Scan() {
			element := index * finalBucket.Size
			copy(finalBucket.Data[element:element+finalBucket.Size], scanners[index].Bytes())
		} else {
			finalBucket.c--
			diff := 0
			for i := range scanners {
				if index == i {
					diff++
				}
				if i < finalBucket.c {
					scanners[i] = scanners[i+diff]
					newElStart := i * finalBucket.Size
					newElEnd := newElStart + finalBucket.Size
					diffEl := diff * finalBucket.Size
					copy(finalBucket.Data[newElStart:newElEnd], finalBucket.Data[newElStart+diffEl:newElEnd+diffEl])
				}
			}
		}
		if finalBucket.c <= 0 {
			break
		}
	}
	for _, handler := range fHandlers {
		handler.Close()
	}
	for _, filename := range files {
		os.Remove(filename)
	}
}

func getSplitter(size int) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if len(data) < size {
			return 0, nil, nil
		}
		return size, data[0:size], nil
	}
}
