package game

import "math"

const pi32 = float32(math.Pi)

func sin32(x float32) float32              { return float32(math.Sin(float64(x))) }
func cos32(x float32) float32              { return float32(math.Cos(float64(x))) }
func atan232(y, x float32) float32         { return float32(math.Atan2(float64(y), float64(x))) }
func sqrt32(x float32) float32             { return float32(math.Sqrt(float64(x))) }
func abs32(x float32) float32              { return float32(math.Abs(float64(x))) }
func remainder32(x, y float32) float32     { return float32(math.Remainder(float64(x), float64(y))) }
