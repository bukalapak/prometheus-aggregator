package main

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type sampleParserState int

const (
	sampleParserStateSearching sampleParserState = iota + 1
	sampleParserStateSample

	sampleParserLabelsSeparator         = ";"
	sampleParserHistogramDefSeparator   = ";"
	sampleParserLabelFromValueSeparator = "="
	sampleParserSamplePartsSeparator    = "|"
)

var (
	labelNameREPart          = `[a-zA-Z_][a-zA-Z0-9_]*`
	labelValueREPart         = `[^;|]+`
	labelWithValueREPart     = labelNameREPart + sampleParserLabelFromValueSeparator + labelValueREPart
	sampleParserLabelsREPart = `(` + labelWithValueREPart + `(` + sampleParserLabelsSeparator + labelWithValueREPart + `)*)`

	sampleParserSharedLabelsLineRE = regexp.MustCompile(`^` + sampleParserLabelsREPart + `$`)

	metricNameREPart         = `[a-zA-Z_:][a-zA-Z0-9_:]+`
	sampleKindREPart         = `(c|g|hl|h)`
	sampleHistogramDefREPart = `[0-9.]+(;[0-9.]+)*`
	// TODO(szpakas): tighter regexp with only one decimal separator
	sampleValueREPart            = `[0-9.]+`
	sampleParserSampleLineREPart = `^` +
		metricNameREPart + `\|` +
		sampleKindREPart + `\|` +
		`(` + sampleHistogramDefREPart + `\|)?` + // optional
		`(` + sampleParserLabelsREPart + `\|)?` + // optional
		sampleValueREPart +
		`$`
	sampleHistogramDefRE = regexp.MustCompile(`^` + sampleHistogramDefREPart + `$`)
	sampleParserSampleLineRE = regexp.MustCompile(sampleParserSampleLineREPart)
)

// parseSample reads a single sample/s description and converts it to set of samples
func parseSample(r io.Reader) ([]*sample, error) {
	var out []*sample

	scanner := bufio.NewScanner(r)

	kindMapper := func(symbol string) sampleKind {
		switch symbol {
		case string(sampleCounter):
			return sampleCounter
		case string(sampleGauge):
			return sampleGauge
		case string(sampleHistogram):
			return sampleHistogram
		case string(sampleHistogramLinear):
			return sampleHistogramLinear
		}
		return sampleUnknown
	}

	labelsMapper := func(s string, out map[string]string) {
		for _, labelWithValue := range strings.Split(s, sampleParserLabelsSeparator) {
			// expecting always 2 values. It's enforced by earlier regexp check
			labelWithValueSlice := strings.SplitN(labelWithValue, sampleParserLabelFromValueSeparator, 2)
			out[labelWithValueSlice[0]] = labelWithValueSlice[1]
		}
	}

	isSampleLine := func(s string) bool {
		return sampleParserSampleLineRE.MatchString(s)
	}

	isHistogramDef := func(s string) bool {
		return sampleHistogramDefRE.MatchString(s)
	}

	parseSampleLine := func(s string, sharedLabels map[string]string) *sample {
		samplePartsSlice := strings.Split(s, sampleParserSamplePartsSeparator)

		labels := make(map[string]string)
		for k, v := range sharedLabels {
			labels[k] = v
		}

		smp := sample{
			name:   samplePartsSlice[0],
			kind:   kindMapper(samplePartsSlice[1]),
			labels: labels,
		}
		smp.value, _ = strconv.ParseFloat(samplePartsSlice[len(samplePartsSlice)-1], 10)

		switch smp.kind {
		case sampleHistogramLinear, sampleHistogram:
			// account for histogramDef
			if len(samplePartsSlice) == 5 {
				smp.histogramDef = strings.Split(samplePartsSlice[2], sampleParserHistogramDefSeparator)
				labelsMapper(samplePartsSlice[3], smp.labels)
			} else {
				if isHistogramDef(samplePartsSlice[2]) {
					smp.histogramDef = strings.Split(samplePartsSlice[2], sampleParserHistogramDefSeparator)
				} else {
					labelsMapper(samplePartsSlice[2], smp.labels)
				}
			}
		default:
			if len(samplePartsSlice) == 4 {
				labelsMapper(samplePartsSlice[2], smp.labels)
			}
		}

		return &smp
	}

	state := sampleParserStateSearching
	sharedLabels := make(map[string]string)

	for scanner.Scan() {
		switch state {
		case sampleParserStateSearching:
			if sampleParserSharedLabelsLineRE.MatchString(scanner.Text()) {
				sharedLabels = make(map[string]string) // reset
				labelsMapper(scanner.Text(), sharedLabels)
				state = sampleParserStateSample
				continue
			}
			text := scanner.Text()

			if isSampleLine(text) {
				out = append(out, parseSampleLine(scanner.Text(), sharedLabels))
				continue
			}

		case sampleParserStateSample:
			if isSampleLine(scanner.Text()) {
				out = append(out, parseSampleLine(scanner.Text(), sharedLabels))
				continue
			}
		}
	}

	return out, nil
}
