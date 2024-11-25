package main

import (
	"bufio"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveapp"
)

//go:embed style.css
var styleCSS []byte

// CLI flags
var (
	numBuckets = flag.Int("b", 10, "initial number of buckets")
	minValue   = flag.Float64("min", math.NaN(), "minimum value (auto if not specified)")
	maxValue   = flag.Float64("max", math.NaN(), "maximum value (auto if not specified)")
)

var AppClient = waveapp.MakeClient(waveapp.AppOpts{
	CloseOnCtrlC: true,
	GlobalStyles: styleCSS,
})

type HistogramProps struct {
	Values     []float64 `json:"values"`
	NumBuckets int       `json:"numBuckets"`
	MinValue   *float64  `json:"minValue"`
	MaxValue   *float64  `json:"maxValue"`
}

type HistogramBucket struct {
	Start  float64
	End    float64
	Count  int
	Height int
}

func calcStats(values []float64) (mean, median, stddev float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Calculate median
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		median = sorted[len(sorted)/2]
	}

	// Calculate standard deviation
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	stddev = math.Sqrt(sumSquares / float64(len(values)))

	return mean, median, stddev
}

var Histogram = waveapp.DefineComponent[HistogramProps](AppClient, "Histogram",
	func(ctx context.Context, props HistogramProps) any {
		if len(props.Values) == 0 {
			return vdom.H("div", map[string]any{
				"className": "histogram-empty",
			}, "Waiting for data...")
		}

		// Calculate data range and stats
		min, max := props.Values[0], props.Values[0]
		for _, v := range props.Values {
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		mean, median, stddev := calcStats(props.Values)

		// Use provided min/max if set
		bucketMin := min
		if props.MinValue != nil {
			bucketMin = *props.MinValue
		}
		bucketMax := max
		if props.MaxValue != nil {
			bucketMax = *props.MaxValue
		}

		// Create evenly spaced buckets
		bucketSize := (bucketMax - bucketMin) / float64(props.NumBuckets)
		buckets := make([]HistogramBucket, props.NumBuckets)

		// Initialize bucket ranges
		for i := range buckets {
			buckets[i].Start = bucketMin + float64(i)*bucketSize
			buckets[i].End = buckets[i].Start + bucketSize
		}

		// Fill buckets
		maxCount := 0
		for _, v := range props.Values {
			bucketValue := v
			if props.MinValue != nil && v < *props.MinValue {
				bucketValue = *props.MinValue
			}
			if props.MaxValue != nil && v > *props.MaxValue {
				bucketValue = *props.MaxValue
			}

			bucketIdx := int((bucketValue - bucketMin) / bucketSize)
			if bucketIdx >= len(buckets) {
				bucketIdx = len(buckets) - 1
			}
			if bucketIdx >= 0 && bucketIdx < len(buckets) {
				buckets[bucketIdx].Count++
				if buckets[bucketIdx].Count > maxCount {
					maxCount = buckets[bucketIdx].Count
				}
			}
		}

		// Normalize heights
		const maxHeight = 20
		for i := range buckets {
			if maxCount > 0 {
				buckets[i].Height = (buckets[i].Count * maxHeight) / maxCount
			}
		}

		return vdom.H("div", map[string]any{
			"className": "histogram",
		},
			// Stats header
			vdom.H("div", map[string]any{
				"className": "histogram-stats",
			},
				"Count: ", len(props.Values),
				" | Range: ", strconv.FormatFloat(min, 'f', 2, 64),
				" - ", strconv.FormatFloat(max, 'f', 2, 64),
				" | Mean: ", strconv.FormatFloat(mean, 'f', 2, 64),
				" | Median: ", strconv.FormatFloat(median, 'f', 2, 64),
				" | StdDev: ", strconv.FormatFloat(stddev, 'f', 2, 64),
			),

			// Single scrolling container for both bars and labels
			vdom.H("div", map[string]any{
				"className": "histogram-scroll-container",
			},
				// Container for all columns and final label
				vdom.H("div", map[string]any{
					"className": "histogram-content",
				},
					// Generate columns (bars + labels)
					vdom.ForEachIdx(buckets, func(bucket HistogramBucket, idx int) any {
						return vdom.H("div", map[string]any{
							"key":       idx,
							"className": "histogram-column",
						},
							// Count label (visible on hover)
							vdom.H("div", map[string]any{
								"className": "count-label",
							}, strconv.Itoa(bucket.Count)),

							// Bar
							vdom.H("div", map[string]any{
								"className": vdom.Classes(
									"bar",
									vdom.If(bucket.Count == 0, "empty"),
								),
								"style": map[string]any{
									"height": func() string {
										if bucket.Count == 0 {
											return "1px"
										}
										return fmt.Sprintf("%dpx", bucket.Height*8)
									}(),
								},
							}),

							// Label for left boundary
							vdom.H("div", map[string]any{
								"className": "x-label",
							}, strconv.FormatFloat(bucket.Start, 'f', 1, 64)),
						)
					}),

					// Final label for rightmost boundary
					vdom.H("div", map[string]any{
						"className": "histogram-final-label",
					},
						vdom.H("div", map[string]any{
							"className": "x-label",
						}, strconv.FormatFloat(bucketMax, 'f', 1, 64)),
					),
				),
			),
		)
	},
)

var App = waveapp.DefineComponent(AppClient, "App",
	func(ctx context.Context, _ any) any {
		// Initialize with CLI values
		initialMin := (*float64)(nil)
		if !math.IsNaN(*minValue) {
			initialMin = minValue
		}
		initialMax := (*float64)(nil)
		if !math.IsNaN(*maxValue) {
			initialMax = maxValue
		}

		values, _, setValuesFn := vdom.UseStateWithFn(ctx, []float64{})
		numBuckets, setNumBuckets := vdom.UseState(ctx, *numBuckets)
		minValue, setMinValue := vdom.UseState(ctx, (*float64)(initialMin))
		maxValue, setMaxValue := vdom.UseState(ctx, (*float64)(initialMax))

		vdom.UseEffect(ctx, func() func() {
			done := make(chan bool)

			go func() {
				defer close(done)
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" {
						continue
					}

					if num, err := strconv.ParseFloat(line, 64); err == nil {
						setValuesFn(func(currentValues []float64) []float64 {
							newValues := make([]float64, len(currentValues)+1)
							copy(newValues, currentValues)
							newValues[len(currentValues)] = num
							return newValues
						})
						AppClient.SendAsyncInitiation()
					}
				}
			}()

			return func() {
				<-done
			}
		}, []any{})

		return vdom.H("div", map[string]any{
			"className": "app",
		},
			vdom.H("div", map[string]any{
				"className": "controls",
			},
				vdom.H("div", map[string]any{
					"className": "control-group",
				},
					vdom.H("label", nil, "Number of buckets: "),
					vdom.H("input", map[string]any{
						"type":  "text",
						"value": numBuckets,
						"onChange": func(e vdom.VDomEvent) {
							if n, err := strconv.Atoi(e.TargetValue); err == nil && n >= 2 && n <= 100 {
								setNumBuckets(n)
							} else {
								setNumBuckets(10)
							}
						},
					}),
				),

				vdom.H("div", map[string]any{
					"className": "control-group",
				},
					vdom.H("label", nil, "Min value: "),
					vdom.H("input", map[string]any{
						"type": "text",
						"value": func() string {
							if minValue != nil {
								return strconv.FormatFloat(*minValue, 'f', -1, 64)
							}
							return ""
						}(),
						"placeholder": "auto",
						"onChange": func(e vdom.VDomEvent) {
							if e.TargetValue == "" {
								setMinValue(nil)
							} else if val, err := strconv.ParseFloat(e.TargetValue, 64); err == nil {
								setMinValue(&val)
							}
						},
					}),
					vdom.If(minValue != nil,
						vdom.H("button", map[string]any{
							"onClick":   func() { setMinValue(nil) },
							"className": "clear-btn",
						}, "×"),
					),
				),

				vdom.H("div", map[string]any{
					"className": "control-group",
				},
					vdom.H("label", nil, "Max value: "),
					vdom.H("input", map[string]any{
						"type": "text",
						"value": func() string {
							if maxValue != nil {
								return strconv.FormatFloat(*maxValue, 'f', -1, 64)
							}
							return ""
						}(),
						"placeholder": "auto",
						"onChange": func(e vdom.VDomEvent) {
							if e.TargetValue == "" {
								setMaxValue(nil)
							} else if val, err := strconv.ParseFloat(e.TargetValue, 64); err == nil {
								setMaxValue(&val)
							}
						},
					}),
					vdom.If(maxValue != nil,
						vdom.H("button", map[string]any{
							"onClick":   func() { setMaxValue(nil) },
							"className": "clear-btn",
						}, "×"),
					),
				),
			),
			Histogram(HistogramProps{
				Values:     values,
				NumBuckets: numBuckets,
				MinValue:   minValue,
				MaxValue:   maxValue,
			}),
		)
	},
)

func main() {
	// Register WaveApp's flags first
	AppClient.RegisterDefaultFlags()

	// Parse our flags
	flag.Parse()

	if *numBuckets < 2 {
		fmt.Fprintf(os.Stderr, "Number of buckets must be at least 2\n")
		os.Exit(1)
	}
	if *numBuckets > 100 {
		fmt.Fprintf(os.Stderr, "Number of buckets must be at most 100\n")
		os.Exit(1)
	}

	AppClient.RunMain()
}
