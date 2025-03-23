package common

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)

func DecodeLimit(limitString *string, verbose *bool) uint64 {
	decoder := regexp.MustCompile(`([0-9_]+)([MGTPE]*)`)
	pieces := decoder.FindStringSubmatch(*limitString)
	lx, err := strconv.ParseInt(strings.Replace(pieces[1], "_", "", -1), 10, 64)
	limit := uint64(lx)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range pieces[2] {
		switch s {
		case 'M':
			limit *= 1_000_000
		case 'G':
			limit *= 1_000_000_000
		case 'T':
			limit *= 1_000_000_000_000
		case 'P':
			limit *= 1_000_000_000_000_000
		case 'E':
			limit *= 1_000_000_000_000_000_000
		default:
			log.Fatalf(`Unrecognized limit format '%c' from "%s", can't happen`, s, pieces[2])
		}
	}
	if *verbose {
		if limit >= 1_000_000_000_000_000 {
			log.Printf(`Limit: %.1fP`, float64(limit)/1e15)
		} else if limit >= 1_000_000_000_000 {
			log.Printf(`Limit: %.1fT`, float64(limit)/1e12)
		} else if limit >= 1_000_000_000 {
			log.Printf(`Limit: %.1fG`, float64(limit)/1e9)
		} else if limit >= 1_000_000 {
			log.Printf(`Limit: %.1fM`, float64(limit)/1e6)
		} else {
			log.Printf("Limit: %d", limit)
		}
	}
	return limit
}
