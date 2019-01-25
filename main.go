package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/go-yaml/yaml"
)

type Language struct {
	Color string `yaml:"color"`
}

type LanguageColor struct {
	Name  string
	Color LAB
}

type LanguageColorDiff struct {
	Name string
	Diff float64
}

func main() {
	var renderHtml bool
	var diffThreshold float64
	var yamlPath string

	flag.BoolVar(&renderHtml, "html", false, "render output as html instead of plaintext")
	flag.StringVar(&yamlPath, "yaml", "languages.yml", "location of language specification file")
	flag.Float64Var(&diffThreshold, "threshold", 10, "threshold for printing color differences")
	flag.Parse()

	content, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		fmt.Printf("%s\nDid you forget `-yaml <language.yml>`?\n", err.Error())
		os.Exit(1)
	}

	var languages map[string]Language

	err = yaml.Unmarshal(content, &languages)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Convert hex-color input map to LAB color scheme array
	tfc := []LanguageColor{}

	for lang, data := range languages {
		rgb, err := HexToRGB(data.Color)
		if err != nil {
			continue
		}

		c := LanguageColor{lang, XYZToLAB(RGBToXYZ(rgb))}
		tfc = append(tfc, c)
	}

	// Compute the difference between all pairs (simulates difference matrix)
	tfcd := map[string][]LanguageColorDiff{}

	for _, lang := range tfc {
		diffs := []LanguageColorDiff{}

		for _, compLang := range tfc {
			if lang.Name == compLang.Name {
				continue
			}
			diffs = append(diffs, LanguageColorDiff{compLang.Name, CIE94Diff(lang.Color, compLang.Color)})
		}

		sort.Slice(diffs, func(i, j int) bool {
			return diffs[i].Diff < diffs[j].Diff
		})

		tfcd[lang.Name] = diffs
	}

	// Print any differences we need below the threshold for the requested languages

	sb := strings.Builder{}

	if renderHtml {
		sb.WriteString(fmt.Sprintf(`
			<!doctype html>
				<table style="margin-bottom:50px">
				<thead>
				<tr>
					<th width="33%%">CIE1994 Parameters</th>
				</tr>
				</thead>
				<tbody>
				<tr>
					<tr><td>X</td><td>%.4f</td></tr>
					<tr><td>Y</td><td>%.4f</td></tr>
					<tr><td>Z</td><td>%.4f</td></tr>
					<tr><td>WL</td><td>%.4f</td></tr>
					<tr><td>WC</td><td>%.4f</td></tr>
					<tr><td>WH</td><td>%.4f</td></tr>
				</tbody>
				</table>

				<table>
				<thead>
				<tr>
					<th width="30%%">Name</th>
					<th width="30%%">Difference (<a href="https://en.wikipedia.org/wiki/Color_difference#CIE94">CIE1994</a>)</th>
					<th width="40%%">Color</th>
				</tr>
				</thead>
				<tbody>`,
			rX, rY, rZ, wL, wC, wH,
		))
	}

	langs := []string{}
	if len(flag.Args()) != 0 {
		langs = flag.Args()
	} else {
		for name, _ := range languages {
			langs = append(langs, name)
		}
		sort.Strings(langs)
	}

	for _, lang := range langs {
		filtered := []LanguageColorDiff{}

		for _, diff := range tfcd[lang] {
			if diff.Diff >= diffThreshold {
				break
			}

			filtered = append(filtered, diff)
		}
		if len(filtered) == 0 {
			continue
		}

		if !renderHtml {
			fmt.Printf("%s: (%s)\n%s\n", lang, languages[lang].Color, strings.Repeat("=", 80))

			for _, diff := range filtered {
				fmt.Printf("%30s: %8.4f (%s)\n", diff.Name, diff.Diff, languages[diff.Name].Color)
			}
		} else {
			sb.WriteString(fmt.Sprintf(`
				<tr style="height:10px">
					<td colspan="3"></td>
				</tr>
				<tr style="height:1px;background-color:black">
					<td colspan="3"></td>
				</tr>
				<tr id="%s">
					<td style="font-weight:bold">%s</td>
					<td></td>
					<td style="background-color:%s"></td>
				</tr>
				<tr style="height:10px">
					<td colspan="3"></td>
				</tr>`,
				lang, lang, languages[lang].Color))

			for _, diff := range filtered {
				sb.WriteString(fmt.Sprintf(`
					<tr>
						<td><a href="#%s">%s</a></td>
						<td>%.4f</td>
						<td style="background-color:%s"></td>
					</tr>`,
					diff.Name, diff.Name, diff.Diff, languages[diff.Name].Color))
			}
		}
	}

	if renderHtml {
		sb.WriteString("</tbody></table>")
		ioutil.WriteFile("output.html", []byte(sb.String()), 0644)
	}
}

// Formulas from http://www.easyrgb.com/en/math.php.

// https://en.wikipedia.org/wiki/RGB_color_space
type RGB struct {
	R uint8
	G uint8
	B uint8
}

// https://en.wikipedia.org/wiki/CIE_1931_color_space
type XYZ struct {
	X float64
	Y float64
	Z float64
}

// https://en.wikipedia.org/wiki/CIELAB_color_space#CIELAB
type LAB struct {
	L float64
	A float64
	B float64
}

func HexToRGB(s string) (RGB, error) {
	if len(s) != 7 {
		return RGB{}, errors.New("expected hex color of form: #RRGGBB")
	}

	b, err := hex.DecodeString(s[1:])
	if err != nil {
		return RGB{}, err
	}

	if len(b) != 3 {
		return RGB{}, errors.New("decoded hex length != 3")
	}

	return RGB{R: b[0], G: b[1], B: b[2]}, nil
}

func RGBToXYZ(c RGB) XYZ {
	Norm := func(n float64) float64 {
		if n > 0.04045 {
			return math.Pow((n+0.055)/1.055, 2.4)
		} else {
			return n / 12.92
		}
	}

	vR := Norm(float64(c.R)/255) * 100
	vG := Norm(float64(c.G)/255) * 100
	vB := Norm(float64(c.B)/255) * 100

	return XYZ{
		X: vR*0.4124 + vG*0.3576 + vB*0.1805,
		Y: vR*0.2126 + vG*0.7152 + vB*0.0722,
		Z: vR*0.0193 + vG*0.1192 + vB*0.9505,
	}
}

/*
   	Reference values of a perfect reflecting diffuser.

   Observer					10° (CIE 1964)							   Note

   Illuminant        X10        	  Y10        	 Z10
   A				111.144        	100.000        	35.200        	Incandescent/tungsten
   B        		99.178;        	100.000        	84.3493        	Old direct sunlight at noon
   C        		97.285        	100.000        	116.145        	Old daylight
   D50        		96.720        	100.000        	81.427        	ICC profile PCS
   D55        		95.799        	100.000        	90.926        	Mid-morning daylight
   D65        		94.811        	100.000        	107.304        	Daylight, sRGB, Adobe-RGB
   D75        		94.416        	100.000        	120.641        	North sky daylight
   E        		100.000        	100.000        	100.000        	Equal energy
   F1        		94.791        	100.000        	103.191        	Daylight Fluorescent
   F2        		103.280        	100.000        	69.026        	Cool fluorescent
   F3        		108.968        	100.000        	51.965        	White Fluorescent
   F4        		114.961        	100.000        	40.963        	Warm White Fluorescent
   F5        		93.369        	100.000        	98.636        	Daylight Fluorescent
   F6        		102.148        	100.000        	62.074        	Lite White Fluorescent
   F7        		95.792        	100.000        	107.687        	Daylight fluorescent, D65 simulator
   F8        		97.115        	100.000        	81.135        	Sylvania F40, D50 simulator
   F9        		102.116        	100.000        	67.826        	Cool White Fluorescent
   F10        		99.001        	100.000        	83.134        	Ultralume 50, Philips TL85
   F11        		103.866        	100.000        	65.627        	Ultralume 40, Philips TL84
   F12        		111.428        	100.000        	40.353        	Ultralume 30, Philips TL83
*/
const rX = 93.369
const rY = 100.000
const rZ = 98.636

func XYZToLAB(c XYZ) LAB {
	Norm := func(n float64) float64 {
		if n > 0.008856 {
			return math.Pow(n, 1.0/3.0)
		} else {
			return (7.787 * n) + (16.0 / 116.0)
		}
	}

	vX := Norm(c.X / rX)
	vY := Norm(c.Y / rY)
	vZ := Norm(c.Z / rZ)

	return LAB{
		L: 116*vY - 16,
		A: 500 * (vX - vY),
		B: 200 * (vY - vZ),
	}
}

// Weighting factors
const wL = 1.0
const wC = 1.0
const wH = 1.0

func CIE94Diff(c1, c2 LAB) float64 {
	xC1 := math.Sqrt(c1.A*c1.A + c1.B*c1.B)
	xC2 := math.Sqrt(c2.A*c2.A + c2.B*c2.B)

	xDL := c2.L - c1.L
	xDC := xC2 - xC1
	xDE := math.Sqrt(((c1.L - c2.L) * (c1.L - c2.L)) +
		((c1.A - c2.A) * (c1.A - c2.A)) +
		((c1.B - c2.B) * (c1.B - c2.B)))
	xDH := (xDE * xDE) - (xDL * xDL) - (xDC * xDC)

	if xDH > 0 {
		xDH = math.Sqrt(xDH)
	} else {
		xDH = 0
	}

	xSC := 1 + (0.045 * xC1)
	xSH := 1 + (0.015 * xC1)

	xDL /= wL
	xDC /= wC * xSC
	xDH /= wH * xSH

	return math.Sqrt(xDL*xDL + xDC*xDC + xDH*xDH)
}
