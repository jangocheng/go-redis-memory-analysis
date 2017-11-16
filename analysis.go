package gorma

import (
	"strings"
	"fmt"
	"strconv"
	. "github.com/hhxsv5/go-redis-memory-analysis/models"
)

type Report struct {
	Key         string
	Count       uint64
	Size        uint64
	NeverExpire uint64
	AvgTtl      uint64
}

type Analysis struct {
	redis   *RedisClient
	Reports map[uint64]map[string]Report
}

func NewAnalysis(redis *RedisClient) (*Analysis) {
	return &Analysis{redis, map[uint64]map[string]Report{}}
}

func (analysis *Analysis) Start(delimiters []string, limit uint64) {

	match := "*[" + strings.Join(delimiters, "") + "]*"
	databases, _ := analysis.redis.GetDatabases()

	var cursor uint64
	var r Report
	var f float64
	var ttl int64
	var length uint64

	for db, _ := range databases {
		cursor = 0

		keys, _ := analysis.redis.Scan(&cursor, match, 3000)

		fd, fp, tmp, nk := "", 0, 0, ""
		for _, key := range keys {
			fd, fp, tmp, nk, ttl = "", 0, 0, "", 0
			for _, delimiter := range delimiters {
				tmp = strings.Index(key, delimiter)
				if tmp != -1 && (tmp < fp || fp == 0) {
					fd, fp = delimiter, tmp
				}
			}


			if fp == 0 {
				continue
			}

			if _, ok := analysis.Reports[db]; !ok {
				analysis.Reports[db] = map[string]Report{}
			}


			nk = key[0:fp] + fd + "*"

			if _, ok := analysis.Reports[db][nk]; ok {
				r = analysis.Reports[db][nk]
			} else {
				r = Report{nk, 1, 0, 0, 0}
			}

			ttl, _ = analysis.redis.Ttl(key)

			switch ttl {
			case -2:
				continue
			case -1:
				r.NeverExpire++
			default:
				f = float64(r.AvgTtl*(r.Count-r.NeverExpire)+uint64(ttl)) / float64(r.Count+1)
				ttl, _ := strconv.ParseUint(fmt.Sprintf("%0.0f", f), 10, 64)
				r.AvgTtl = ttl
				r.Count++
			}

			length, _ = analysis.redis.SerializedLength(nk)
			r.Size += length

			analysis.Reports[db][nk] = r
		}
	}
}
