package followsvc

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
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

// plotJSON is a general structure holding plot type and unparse arguments for that type
type plotJSON struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

// linePlotJSON is a structure holding arguments for Line and LogLine
type linePlotJSON struct {
	P0, P1 geometry.Point
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

// oggsetPlotJSON is a structure holding arguments for Min and Max
type minMaxPlotJSON struct {
	Plots []plotJSON
}

func parsePlotMap(plot map[string]any) (geometry.Plot, error) {
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
	// 	return geometry.NewProtector(parsedPlot), nil
	// }

	return parsedPlot, nil
}

func parsePlot(pj plotJSON) (geometry.Plot, error) {
	args := pj.Args

	switch pj.Type {
	case KEY_LINE:
		lineJSON := linePlotJSON{}
		if err := json.Unmarshal(args, &lineJSON); err != nil {
			return nil, err
		}

		return geometry.NewLine(lineJSON.P0, lineJSON.P1)
	case KEY_LINE_LOG:
		lineJSON := linePlotJSON{}
		if err := json.Unmarshal(args, &lineJSON); err != nil {
			return nil, err
		}

		return geometry.NewLogLine(lineJSON.P0, lineJSON.P1)
	case KEY_OFFSET_ABSOLUTE:
		offsetJSON := offsetPlotJSON{}
		if err := json.Unmarshal(args, &offsetJSON); err != nil {
			return nil, err
		}

		plotToOffset, err := parsePlot(offsetJSON.Plot)
		if err != nil {
			return nil, err
		}

		return geometry.NewOffsetPlot(plotToOffset, geometry.NewAbsoluteOffset(offsetJSON.Value)), nil
	case KEY_OFFSET_PERCENTAGE:
		offsetJSON := offsetPlotJSON{}
		if err := json.Unmarshal(args, &offsetJSON); err != nil {
			return nil, err
		}

		plotToOffset, err := parsePlot(offsetJSON.Plot)
		if err != nil {
			return nil, err
		}

		return geometry.NewOffsetPlot(plotToOffset, geometry.NewPercentageOffset(offsetJSON.Value)), nil
	case KEY_LIMIT:
		limitJSON := limitPlotJSON{}
		if err := json.Unmarshal(args, &limitJSON); err != nil {
			return nil, err
		}

		plotToLimit, err := parsePlot(limitJSON.Plot)
		if err != nil {
			return nil, err
		}

		return geometry.NewLimit(plotToLimit, limitJSON.Since, limitJSON.Until), nil
	case KEY_MIN:
		minmaxJSON := minMaxPlotJSON{}
		if err := json.Unmarshal(args, &minmaxJSON); err != nil {
			return nil, err
		}

		plotsToMin := []geometry.Plot{}
		for _, pjMin := range minmaxJSON.Plots {
			plotToMin, err := parsePlot(pjMin)
			if err != nil {
				return nil, err
			}

			plotsToMin = append(plotsToMin, plotToMin)
		}

		return geometry.NewMin(plotsToMin)
	case KEY_MAX:
		minmaxJSON := minMaxPlotJSON{}
		if err := json.Unmarshal(args, &minmaxJSON); err != nil {
			return nil, err
		}

		plotsToMin := []geometry.Plot{}
		for _, pjMin := range minmaxJSON.Plots {
			plotToMin, err := parsePlot(pjMin)
			if err != nil {
				return nil, err
			}

			plotsToMin = append(plotsToMin, plotToMin)
		}

		return geometry.NewMax(plotsToMin)
	}

	return nil, fmt.Errorf("unknown plot name %s", pj.Type)
}
