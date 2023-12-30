package geometry

// import (
// 	"fmt"
// 	"time"
// )

// // Shape is a sequence of lines
// type Shape struct {
// 	Lines []*Line
// }

// // NewShape is a Shape constructor, it accepts slice of points which are then sorted by time and connected using lines.
// // If extendLeft is true then the first line extends indefinitely to the left.
// // if extendRight is true then the last line extends indefinitely to the right.
// func NewShape(points []Point, extendLeft, extendRight bool) (*Shape, error) {
// 	if len(points) < 3 {
// 		return nil, fmt.Errorf("at least 3 points are required to create a shape, got: %d", len(points))
// 	}
// 	points = sortPoints(points...)

// 	lines := []*Line{}

// 	for i := 0; i < len(points)-1; i++ {
// 		j := i + 1

// 		line, err := NewLine(points[i], points[j], i == 0 && extendLeft, i == len(points)-1 && extendRight)
// 		if err != nil {
// 			return nil, fmt.Errorf("error creating line between points %d and %d: %w", i, j, err)
// 		}

// 		lines = append(lines, line)
// 	}

// 	return &Shape{Lines: lines}, nil
// }

// func (s *Shape) At(t time.Time) (float64, bool) {
// 	for _, line := range s.Lines {
// 		v, ok := line.At(t)
// 		if !ok {
// 			continue
// 		}

// 		return v, true
// 	}

// 	return 0, false
// }

// // LogShape is a Shape that uses LogLines instead of Lines
// type LogShape struct {
// 	Lines []*LogLine
// }

// // NewShape requires at least 3 points
// func NewLogShape(points []Point, extendLeft, extendRight bool) (*LogShape, error) {
// 	if len(points) < 3 {
// 		return nil, fmt.Errorf("at least 3 points are required to create a shape, got: %d", len(points))
// 	}
// 	points = sortPoints(points...)

// 	lines := []*LogLine{}

// 	for i := 0; i < len(points)-1; i++ {
// 		j := i + 1

// 		line, err := NewLogLine(points[i], points[j], i == 0 && extendLeft, i == len(points)-1 && extendRight)
// 		if err != nil {
// 			return nil, fmt.Errorf("error creating line between points %d and %d: %w", i, j, err)
// 		}

// 		lines = append(lines, line)
// 	}

// 	return &LogShape{Lines: lines}, nil
// }

// func (s *LogShape) At(t time.Time) (float64, bool) {
// 	for _, line := range s.Lines {
// 		v, ok := line.At(t)
// 		if !ok {
// 			continue
// 		}

// 		return v, true
// 	}

// 	return 0, false
// }
