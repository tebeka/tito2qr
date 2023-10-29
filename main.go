package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"slices"

	"github.com/skip2/go-qrcode"
)

var usage = `usage: %s [options]
Encode exported ti.to attendee CSV to QR images with VCARD information.

`

var vcardTemplate = `BEGIN:VCARD
VERSION:4.0
FN:%[1]s %[2]s
N:%[2]s;%[1]s
EMAIL;TYPE=work:%[3]s
END:VCARD
`

func main() {
	var inFile string

	flag.StringVar(&inFile, "input", "", "input file (ti.to exported CSV)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, path.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()
	log.SetFlags(0) // no time prefix

	var r io.Reader = os.Stdin
	if inFile != "" {
		file, err := os.Open(inFile)
		if err != nil {
			log.Fatalf("error: %s", err)
		}
		defer file.Close()
		r = file
	}
	fileName := inFile
	if fileName == "" {
		fileName = "<stdin>"
	}

	rdr := csv.NewReader(r)
	header, err := rdr.Read()
	if err != nil {
		log.Fatalf("error: %q - empty file? (%s)", fileName, err)
	}

	firstIdx := slices.Index(header, "Ticket First Name")
	lastIdx := slices.Index(header, "Ticket Last Name")
	emailIdx := slices.Index(header, "Ticket Email")

	if firstIdx == -1 || lastIdx == -1 || emailIdx == -1 {
		log.Fatalf("error: %q - bad header", fileName)
	}
	maxIdx := max(firstIdx, lastIdx, emailIdx)

	lnum := 1
	for {
		lnum++
		record, err := rdr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalf("error: %q:%d: %s", fileName, lnum, err)
		}

		if len(record) <= maxIdx {
			log.Fatalf("error: %q:%d: record too short", fileName, lnum)
		}

		first, last, email := record[firstIdx], record[lastIdx], record[emailIdx]
		if first == "" && last == "" {
			continue
		}

		vcard := fmt.Sprintf(vcardTemplate, first, last, email)
		outFile := fmt.Sprintf("%d-%s-%s.png", lnum, first, last)
		if err := qrcode.WriteFile(vcard, qrcode.Medium, 256, outFile); err != nil {
			log.Fatalf("error: %q:%d: can't write QR to %q - %s", fileName, lnum, outFile, err)
		}
	}
}
