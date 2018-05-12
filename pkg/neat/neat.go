// Copyright © 2018 Matthias Diester
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package neat

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/HeavyWombat/dyff/pkg/bunt"
	yaml "gopkg.in/yaml.v2"
)

// DefaultColorSchema is a prepared usable color schema for the neat output
// processor which is loosly based upon the colors used by Atom
var DefaultColorSchema = map[string]bunt.Color{
	"keyColor":           bunt.IndianRed,
	"indentLineColor":    bunt.Color(0x00242424),
	"scalarDefaultColor": bunt.PaleGreen,
	"boolColor":          bunt.Moccasin,
	"floatColor":         bunt.Orange,
	"intColor":           bunt.MediumPurple,
	"multiLineTextColor": bunt.Aquamarine,
	"nullColor":          bunt.DarkOrange,
	"emptyStructures":    bunt.PaleGoldenrod,
}

// OutputProcessor provides the functionality to output neat YAML strings using
// colors and text emphasis
type OutputProcessor struct {
	data           *bytes.Buffer
	out            *bufio.Writer
	colorSchema    *map[string]bunt.Color
	useIndentLines bool
	boldKeys       bool
}

// ToYAMLString marshals the provided object into YAML with text decorations
// and is basically just a convenience function to create the output processor
// and call its ToString function.
func ToYAMLString(obj interface{}) (string, error) {
	return NewOutputProcessor(true, true, &DefaultColorSchema).ToString(obj)
}

// NewOutputProcessor creates a new output processor including the required
// internals using the provided preferences
func NewOutputProcessor(useIndentLines bool, boldKeys bool, colorSchema *map[string]bunt.Color) *OutputProcessor {
	bytesBuffer := &bytes.Buffer{}
	writer := bufio.NewWriter(bytesBuffer)

	return &OutputProcessor{
		data:           bytesBuffer,
		out:            writer,
		useIndentLines: useIndentLines,
		boldKeys:       boldKeys,
		colorSchema:    colorSchema,
	}
}

// ToString processes the provided input object and tries to neatly output it as
// human readable YAML honoring the preferences provided to the output processor
func (p *OutputProcessor) ToString(obj interface{}) (string, error) {
	if err := p.neat("", false, obj); err != nil {
		return "", err
	}

	p.out.Flush()
	return p.data.String(), nil
}

func (p *OutputProcessor) colorize(text string, colorName string) string {
	if p.colorSchema != nil {
		if value, ok := (*p.colorSchema)[colorName]; ok {
			return bunt.Colorize(text, value)
		}
	}

	return text
}

func (p *OutputProcessor) neat(prefix string, skipIndentOnFirstLine bool, obj interface{}) error {
	switch obj.(type) {
	case yaml.MapSlice:
		if err := p.neatMapSlice(prefix, skipIndentOnFirstLine, obj.(yaml.MapSlice)); err != nil {
			return err
		}

	case []interface{}:
		if err := p.neatSlice(prefix, skipIndentOnFirstLine, obj.([]interface{})); err != nil {
			return err
		}

	case []yaml.MapSlice:
		if err := p.neatMapSliceSlice(prefix, skipIndentOnFirstLine, obj.([]yaml.MapSlice)); err != nil {
			return err
		}

	default:
		if err := p.neatScalar(prefix, skipIndentOnFirstLine, obj); err != nil {
			return err
		}
	}

	return nil
}

func (p *OutputProcessor) neatMapSlice(prefix string, skipIndentOnFirstLine bool, mapslice yaml.MapSlice) error {
	for i, mapitem := range mapslice {
		if !skipIndentOnFirstLine || i > 0 {
			p.out.WriteString(prefix)
		}

		keyString := fmt.Sprintf("%v:", mapitem.Key)
		if p.boldKeys {
			keyString = bunt.Style(keyString, bunt.Bold)
		}

		p.out.WriteString(p.colorize(keyString, "keyColor"))

		switch mapitem.Value.(type) {
		case yaml.MapSlice:
			if len(mapitem.Value.(yaml.MapSlice)) == 0 {
				p.out.WriteString(" ")
				p.out.WriteString(p.colorize("{}", "emptyStructures"))
				p.out.WriteString("\n")

			} else {
				p.out.WriteString("\n")
				if err := p.neatMapSlice(prefix+p.prefixAdd(), false, mapitem.Value.(yaml.MapSlice)); err != nil {
					return err
				}
			}

		case []interface{}:
			if len(mapitem.Value.([]interface{})) == 0 {
				p.out.WriteString(" ")
				p.out.WriteString(p.colorize("[]", "emptyStructures"))
				p.out.WriteString("\n")
			} else {
				p.out.WriteString("\n")
				if err := p.neatSlice(prefix, false, mapitem.Value.([]interface{})); err != nil {
					return err
				}
			}

		default:
			p.out.WriteString(" ")
			if err := p.neatScalar(prefix, false, mapitem.Value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *OutputProcessor) neatSlice(prefix string, skipIndentOnFirstLine bool, list []interface{}) error {
	for _, entry := range list {
		p.out.WriteString(prefix)
		p.out.WriteString(bunt.Style("- ", bunt.Bold))
		if err := p.neat(prefix+p.prefixAdd(), true, entry); err != nil {
			return err
		}
	}

	return nil
}

func (p *OutputProcessor) neatMapSliceSlice(prefix string, skipIndentOnFirstLine bool, list []yaml.MapSlice) error {
	for _, entry := range list {
		p.out.WriteString(prefix)
		p.out.WriteString(bunt.Style("- ", bunt.Bold))
		if err := p.neat(prefix+p.prefixAdd(), true, entry); err != nil {
			return err
		}
	}

	return nil
}

func (p *OutputProcessor) neatScalar(prefix string, skipIndentOnFirstLine bool, obj interface{}) error {
	// Process nil values immediately and return afterwards
	if obj == nil {
		p.out.WriteString(p.colorize("null", "nullColor"))
		p.out.WriteString("\n")
		return nil
	}

	// Any other value: Run through Go YAML marshaller and colorize afterwards
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	color := "scalarDefaultColor"
	switch obj.(type) {
	case bool:
		color = "boolColor"

	case float32, float64:
		color = "floatColor"

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr:
		color = "intColor"
	}

	// Cast byte slice to string, remove trailing newlines, split into lines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(lines) > 1 {
		color = "multiLineTextColor"
	}

	for i, line := range lines {
		if i > 0 {
			p.out.WriteString(prefix)
		}

		p.out.WriteString(p.colorize(line, color))
		p.out.WriteString("\n")
	}

	return nil
}

func (p *OutputProcessor) prefixAdd() string {
	if p.useIndentLines {
		return p.colorize("│ ", "indentLineColor")
	}

	return p.colorize("  ", "indentLineColor")
}
