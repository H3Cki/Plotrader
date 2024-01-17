package geometry

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

const (
	KEY_LINE              = "line"
	KEY_LINE_LOG          = "line_log"
	KEY_OFFSET_ABSOLUTE   = "offset_absolute"
	KEY_OFFSET_PERCENTAGE = "offset_percentage"
	KEY_MIN               = "min"
	KEY_MAX               = "max"
	KEY_LIMIT             = "limit"
)

var formats = []string{
	"Mon 02 Jan'06 15:04",        // TradingView
	"Mon 02 Jan'06",              // TradingView 2
	"2006/01/02 15:04",           // Binance
	"01/02 03:04:05PM '06 -0700", // The reference time, in numerical order.
	"Mon Jan _2 15:04:05 2006",
	"Mon Jan _2 15:04:05 MST 2006",
	"Mon Jan 02 15:04:05 -0700 2006",
	"02 Jan 06 15:04 MST",
	"02 Jan 06 15:04 -0700", // RFC822 with numeric zone
	"Monday, 02-Jan-06 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 -0700", // RFC1123 with numeric zone
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"3:04PM",
	// Handy time stamps.
	"Jan _2 15:04:05",
	"Jan _2 15:04:05.000",
	"Jan _2 15:04:05.000000",
	"Jan _2 15:04:05.000000000",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"15:04:05",
}

func parseTime(s string) (time.Time, error) {
	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time string: %s", s)
}

// plotJSON is a general structure holding plot type and unparse arguments for that type
type plotJSON struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

// linePlotJSON is a structure holding arguments for Line and LogLine
type linePlotJSON struct {
	P0, P1 pointJSON
}

type pointJSON Point

func (p *pointJSON) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Date  string  `json:"date"`
		Price float64 `json:"price"`
	}{}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	time, err := parseTime(tmp.Date)
	if err != nil {
		return err
	}

	p.Date = time
	p.Price = tmp.Price
	return nil
}

// oggsetPlotJSON is a structure holding arguments for AbsoluteOffset and PercentageOffset
type offsetPlotJSON struct {
	Value float64
	Plot  plotJSON
}

// oggsetPlotJSON is a structure holding arguments for Schedule
type limitPlotJSON struct {
	Since, Until time.Time
	Plot         plotJSON
}

func (p *limitPlotJSON) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Since, Until string
		Plot         plotJSON
	}{}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	since, err := parseTime(tmp.Since)
	if err != nil {
		return err
	}

	until, err := parseTime(tmp.Until)
	if err != nil {
		return err
	}

	p.Since = since
	p.Until = until
	p.Plot = tmp.Plot
	return nil
}

// oggsetPlotJSON is a structure holding arguments for Min and Max
type minMaxPlotJSON struct {
	Plots []plotJSON
}

func parsePlotMap(plot map[string]any) (Plot, error) {
	bytes, err := json.Marshal(plot)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling plot")
	}

	pj := plotJSON{}
	if err := json.Unmarshal(bytes, &pj); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling plot")
	}

	parsedPlot, err := parsePlot(pj)
	if err != nil {
		return nil, fmt.Errorf("error parsing plot: %w", err)
	}

	// if protect {
	// 	return NewProtector(parsedPlot), nil
	// }

	return parsedPlot, nil
}

func parsePlot(pj plotJSON) (Plot, error) {
	args := pj.Args

	switch pj.Type {
	case KEY_LINE:
		lineJSON := linePlotJSON{}
		if err := json.Unmarshal(args, &lineJSON); err != nil {
			return nil, err
		}

		return NewLine(Point(lineJSON.P0), Point(lineJSON.P1))
	case KEY_LINE_LOG:
		lineJSON := linePlotJSON{}
		if err := json.Unmarshal(args, &lineJSON); err != nil {
			return nil, err
		}

		return NewLogLine(Point(lineJSON.P0), Point(lineJSON.P1))
	case KEY_OFFSET_ABSOLUTE:
		offsetJSON := offsetPlotJSON{}
		if err := json.Unmarshal(args, &offsetJSON); err != nil {
			return nil, err
		}

		plotToOffset, err := parsePlot(offsetJSON.Plot)
		if err != nil {
			return nil, err
		}

		return NewOffsetPlot(plotToOffset, NewAbsoluteOffset(offsetJSON.Value)), nil
	case KEY_OFFSET_PERCENTAGE:
		offsetJSON := offsetPlotJSON{}
		if err := json.Unmarshal(args, &offsetJSON); err != nil {
			return nil, err
		}

		plotToOffset, err := parsePlot(offsetJSON.Plot)
		if err != nil {
			return nil, err
		}

		return NewOffsetPlot(plotToOffset, NewPercentageOffset(offsetJSON.Value)), nil
	case KEY_LIMIT:
		limitJSON := limitPlotJSON{}
		if err := json.Unmarshal(args, &limitJSON); err != nil {
			return nil, err
		}

		plotToLimit, err := parsePlot(limitJSON.Plot)
		if err != nil {
			return nil, err
		}

		return NewLimit(plotToLimit, limitJSON.Since, limitJSON.Until), nil
	case KEY_MIN:
		minmaxJSON := minMaxPlotJSON{}
		if err := json.Unmarshal(args, &minmaxJSON); err != nil {
			return nil, err
		}

		plotsToMin := []Plot{}
		for _, pjMin := range minmaxJSON.Plots {
			plotToMin, err := parsePlot(pjMin)
			if err != nil {
				return nil, err
			}

			plotsToMin = append(plotsToMin, plotToMin)
		}

		return NewMin(plotsToMin)
	case KEY_MAX:
		minmaxJSON := minMaxPlotJSON{}
		if err := json.Unmarshal(args, &minmaxJSON); err != nil {
			return nil, err
		}

		plotsToMin := []Plot{}
		for _, pjMin := range minmaxJSON.Plots {
			plotToMin, err := parsePlot(pjMin)
			if err != nil {
				return nil, err
			}

			plotsToMin = append(plotsToMin, plotToMin)
		}

		return NewMax(plotsToMin)
	}

	return nil, fmt.Errorf("unknown plot name %s", pj.Type)
}
