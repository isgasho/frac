/*
Missing feature of the Go standard library: parsing and formatting integers as
fractional numeric strings, without any rounding or bignums, by using a fixed
fraction size. Supports arbitrary radixes from 2 to 36.

See `readme.md` for examples.
*/
package frac

import (
	"fmt"
	"unsafe"
)

// Shortcut for `Parse(src, frac, 2)`.
func ParseBin(src string, frac uint) (int64, error) {
	return Parse(src, frac, 2)
}

// Shortcut for `Parse(src, frac, 8)`.
func ParseOct(src string, frac uint) (int64, error) {
	return Parse(src, frac, 8)
}

// Shortcut for `Parse(src, frac, 10)`.
func ParseDec(src string, frac uint) (int64, error) {
	return Parse(src, frac, 10)
}

// Shortcut for `Parse(src, frac, 16)`.
func ParseHex(src string, frac uint) (int64, error) {
	return Parse(src, frac, 16)
}

/*
Parses a string that represents a fractional number, with an optional leading
"+" or "-", into an integer whose value is virtually "multiplied" by the
provided fractional precision.

For example, for `frac = 2, radix = 10`, "123.45" is parsed into the number
12345, while "123.456" is rejected with an error because it exceeds the
allotted precision.

See `readme.md` for examples.
*/
func Parse(src string, frac uint, radix uint) (num int64, err error) {
	if len(src) == 0 {
		return 0, fmt.Errorf(`can't parse empty input as number`)
	}

	if !(radix >= 2 && radix <= 36) {
		return 0, fmt.Errorf(`can't parse %q as number: unsupported radix %v`, src, radix)
	}

	var sign int64 = 1
	var expDigs uint

	const (
		stepSign = iota
		stepMantStart
		stepMant
		stepExpStart
		stepExp
	)
	step := stepSign

	for i, char := range []byte(src) {
		if step == stepSign {
			if char == '+' {
				step = stepMantStart
				continue
			}

			if char == '-' {
				sign = -1
				step = stepMantStart
				continue
			}

			step = stepMantStart
		}

		if step == stepMant && char == '.' {
			step = stepExpStart
			continue
		}

		if step == stepMantStart {
			step = stepMant
		} else if step == stepExpStart {
			step = stepExp
		}

		digit := toDigit(char)
		if digit == unDigit || uint(digit) >= radix {
			return 0, fmt.Errorf(`can't parse %q as number (radix %v, fraction %v): found non-digit character %q`,
				src, radix, frac, runeAt(src, i))
		}

		if step == stepExp {
			expDigs++
			if expDigs > frac {
				if digit == 0 {
					continue
				}
				return 0, fmt.Errorf(`can't parse %q as number (radix %v, fraction %v): exponent exceeds allotted fractional precision`,
					src, radix, frac)
			}
		}

		if num == 0 {
			num = sign * int64(digit)
			continue
		}

		num, err = inc(src, num, radix, sign, digit)
		if err != nil {
			return 0, err
		}
	}

	for expDigs < frac {
		num, err = inc(src, num, radix, sign, 0)
		if err != nil {
			return 0, err
		}
		expDigs++
	}

	if step != stepMant && step != stepExp {
		return 0, fmt.Errorf(`can't parse %q as number (radix %v, fraction %v): unexpected end of input`,
			src, radix, frac)
	}
	return num, nil
}

// Shortcut for `UnmarshalBin(src, frac, 2)`.
func UnmarshalBin(src []byte, frac uint) (int64, error) {
	return Unmarshal(src, frac, 2)
}

// Shortcut for `UnmarshalOct(src, frac, 8)`.
func UnmarshalOct(src []byte, frac uint) (int64, error) {
	return Unmarshal(src, frac, 8)
}

// Shortcut for `UnmarshalDec(src, frac, 10)`.
func UnmarshalDec(src []byte, frac uint) (int64, error) {
	return Unmarshal(src, frac, 10)
}

// Shortcut for `UnmarshalHex(src, frac, 16)`.
func UnmarshalHex(src []byte, frac uint) (int64, error) {
	return Unmarshal(src, frac, 16)
}

// Same as `Parse` but takes a byte slice.
func Unmarshal(src []byte, frac uint, radix uint) (int64, error) {
	return Parse(bytesToMutableString(src), frac, radix)
}

// Shortcut for `Format(num, frac, 2)`.
func FormatBin(num int64, frac uint) string {
	return tryString(Format(num, frac, 2))
}

// Shortcut for `Format(num, frac, 8)`.
func FormatOct(num int64, frac uint) string {
	return tryString(Format(num, frac, 8))
}

// Shortcut for `Format(num, frac, 10)`.
func FormatDec(num int64, frac uint) string {
	return tryString(Format(num, frac, 10))
}

// Shortcut for `Format(num, frac, 16)`.
func FormatHex(num int64, frac uint) string {
	return tryString(Format(num, frac, 16))
}

/*
Formats an integer as a fractional number, virtually "divided" by the given
fractional precision.

For example, for `frac = 2, radix = 10`, the number 12345 is encoded
as "123.45", while the number 12300 is encoded as simply "123".
*/
func Format(num int64, frac uint, radix uint) (string, error) {
	buf, err := Append(nil, num, frac, radix)
	return bytesToMutableString(buf), err
}

// Shortcut for `Append(buf, num, frac, 2)`.
func AppendBin(buf []byte, num int64, frac uint) []byte {
	return tryByteSlice(Append(buf, num, frac, 2))
}

// Shortcut for `Append(buf, num, frac, 8)`.
func AppendOct(buf []byte, num int64, frac uint) []byte {
	return tryByteSlice(Append(buf, num, frac, 8))
}

// Shortcut for `Append(buf, num, frac, 10)`.
func AppendDec(buf []byte, num int64, frac uint) []byte {
	return tryByteSlice(Append(buf, num, frac, 10))
}

// Shortcut for `Append(buf, num, frac, 16)`.
func AppendHex(buf []byte, num int64, frac uint) []byte {
	return tryByteSlice(Append(buf, num, frac, 16))
}

/*
Same as `Format` but appends the resulting text to the provided buffer,
returning the resulting union. When there's an error, the buffer is returned
as-is with no hidden modifications.
*/
func Append(buf []byte, num int64, frac uint, radix uint) ([]byte, error) {
	if !(radix >= 2 && radix <= 36) {
		return buf, fmt.Errorf(`can't encode %v: unsupported radix %v`, num, radix)
	}

	// Up to all bits of the number plus leading "-0.".
	var local [unsafe.Sizeof(num)*8 + 3]byte
	i := len(local)

	var neg bool
	var unum uint64
	if num < 0 {
		neg = true
		unum = uint64(-num)
	} else {
		unum = uint64(num)
	}

	rad := uint64(radix)
	trailing := true
	var digit uint64

	for frac > 0 {
		frac--
		unum, digit = pop(unum, rad)

		if digit == 0 && trailing {
			continue
		}
		trailing = false

		i--
		local[i] = digits[digit]

		if frac == 0 {
			i--
			local[i] = '.'
		}
	}

	for unum >= rad {
		unum, digit = pop(unum, rad)
		i--
		local[i] = digits[digit]
	}

	i--
	local[i] = digits[unum]

	if neg {
		i--
		local[i] = '-'
	}

	return append(buf, local[i:]...), nil
}

const digits = `0123456789abcdefghijklmnopqrstuvwxyz`

func inc(src string, prev int64, radix uint, sign int64, digit byte) (int64, error) {
	next := prev*int64(radix) + sign*int64(digit)
	if prev > 0 && next < prev {
		return 0, fmt.Errorf(`can't parse %q as number: overflow of %T`, src, next)
	}
	if prev < 0 && next > prev {
		return 0, fmt.Errorf(`can't parse %q as number: underflow of %T`, src, next)
	}
	return next, nil
}

const unDigit byte = 255

func toDigit(char byte) byte {
	if char >= '0' && char <= '9' {
		return char - '0'
	}
	char = lower(char)
	if char >= 'a' && char <= 'z' {
		return char - 'a' + 10
	}
	return unDigit
}

func lower(char byte) byte {
	return char | ('a' - 'A')
}

func runeAt(str string, index int) rune {
	for i, char := range str {
		if i == index {
			return char
		}
	}
	panic(fmt.Errorf(`failed to get rune from %q at %v`, str, index))
}

func pop(num uint64, radix uint64) (uint64, uint64) {
	quot := num / radix
	digit := num - quot*radix
	return quot, digit
}

/*
Allocation-free conversion. Reinterprets a byte slice as a string. Borrowed from
the standard library. Reasonably safe.
*/
func bytesToMutableString(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}

func tryString(val string, err error) string {
	if err != nil {
		panic(err)
	}
	return val
}

func tryByteSlice(val []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return val
}