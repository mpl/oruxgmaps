// Copyright 2016 Mathieu "mpl" Lonjaret

package oruxgmaps

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

// Insert reads the original onlinemapsources.xml from r, inserts the Google
// Maps definition in the xml schema, and returns the result.
func Insert(r io.Reader) ([]byte, error) {
	lr := io.LimitedReader{
		R: r,
		N: 1 << 20,
	}

	var (
		dataBefore []byte
		dataAfter  []byte
	)

	// Parsing XML with regexp is bad, mkay?
	endOfMapSourcesRxp := regexp.MustCompile("^</onlinemapsources>")
	mapSourceRxp := regexp.MustCompile(`(\s)*<onlinemapsource uid="([0-9]+)">.*`)

	lastuid := 0
	isAfter := false
	sc := bufio.NewScanner(&lr)
	for sc.Scan() {
		l := sc.Bytes()
		if endOfMapSourcesRxp.Match(l) {
			isAfter = true
			dataAfter = append(dataAfter, append(l, '\n')...)
			break
		}
		l = append(l, '\n')
		if isAfter {
			dataAfter = append(dataAfter, l...)
		} else {
			dataBefore = append(dataBefore, l...)
		}
		m := mapSourceRxp.FindSubmatch(l)
		if len(m) == 0 {
			continue
		}
		uid, err := strconv.Atoi(string(m[2]))
		if err != nil {
			return nil, err
		}
		lastuid = uid
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	lastuid++
	gmapsDef := fmt.Sprintf(
		`
<onlinemapsource uid="%d">
<name>Google Maps</name>
<url><![CDATA[http://mt{$s}.google.com/vt/lyrs=m@121&hl={$l}&x={$x}&y={$y}&z={$z}]]></url>
<website><![CDATA[<a href="https://maps.google.com">web site link</a>]]></website>
<minzoom>0</minzoom>
<maxzoom>19</maxzoom>
<projection>MERCATORESFERICA</projection>
<servers>0,1,2,3</servers>
<httpparam name=""></httpparam>
<cacheable>1</cacheable>
<downloadable>1</downloadable>
<maxtilesday>0</maxtilesday>
<maxthreads>0</maxthreads>
<xop></xop>
<yop></yop>
<zop></zop>
<qop></qop>
<sop></sop>
</onlinemapsource>
`, lastuid)

	return append(append(dataBefore, []byte(gmapsDef)...), dataAfter...), nil
}
