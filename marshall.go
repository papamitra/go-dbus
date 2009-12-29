package dbus

import (
	"bytes"
	"strings"
	"encoding/binary"
	"os"
	"container/vector"
	"fmt"
)

func _Align(length int, index int) int {
	switch length {
	case 1:
		return index
	case 2, 4, 8:
		bit := length - 1
		return ^bit & (index + bit)
	}
	// default
	return -1
}

func _AppendAlign(length int, buff *bytes.Buffer) {
	padno := _Align(length, buff.Len()) - buff.Len()
	for i := 0; i < padno; i++ {
		buff.WriteByte(0)
	}
}

func _AppendString(buff *bytes.Buffer, str string) {
	_AppendAlign(4, buff)
	binary.Write(buff, binary.LittleEndian, int32(len(str)))
	buff.Write(strings.Bytes(str))
	buff.WriteByte(0)
}

func _AppendSignature(buff *bytes.Buffer, sig string) {
	_AppendByte(buff, byte(len(sig)))
	buff.Write(strings.Bytes(sig))
	buff.WriteByte(0)
}

func _AppendByte(buff *bytes.Buffer, b byte) { binary.Write(buff, binary.LittleEndian, b) }

func _AppendUint32(buff *bytes.Buffer, ui uint32) {
	_AppendAlign(4, buff)
	binary.Write(buff, binary.LittleEndian, ui)
}

func _AppendInt32(buff *bytes.Buffer, i int32) {
	_AppendAlign(4, buff)
	binary.Write(buff, binary.LittleEndian, i)
}

func _AppendArray(buff *bytes.Buffer, align int, proc func(b *bytes.Buffer)) {
	_AppendAlign(4, buff)
	_AppendAlign(align, buff)
	b := bytes.NewBuffer(buff.Bytes())
	b.Write(strings.Bytes("ABCD")) // "ABCD" will be replaced with array-size.
	pos1 := b.Len()
	proc(b)
	pos2 := b.Len()
	binary.Write(buff, binary.LittleEndian, int32(pos2-pos1))
	buff.Write(b.Bytes()[pos1:pos2])
}

func _AppendValue(buff *bytes.Buffer, sig string, val interface{}) (sigOffset int, e os.Error) {
	if len(sig) == 0 {
		return 0, os.NewError("Invalid Signature")
	}

	e = nil

	switch sig[0] {
	case 'y': // byte
		_AppendByte(buff, val.(byte))
		sigOffset =1

	case 's': // string
		_AppendString(buff, val.(string))
		sigOffset = 1

	case 'u': // uint32
		_AppendUint32(buff, val.(uint32))
		sigOffset = 1

	case 'i': // int32
		_AppendInt32(buff, val.(int32))
		sigOffset = 1

	case 'a': // ary
		sigBlock, _ := _GetSigBlock(sig, 1)
		_AppendArray(buff, 1, func(b *bytes.Buffer) {
			if vec, ok := val.(*vector.Vector); ok && vec != nil {
				for v := range vec.Iter() {
					_AppendValue(b, sigBlock, v)
				}
			}
		})
		sigOffset = 1 + len(sigBlock)

	case '(': // struct FIXME: nested struct not support
		_AppendAlign(8, buff)
		structSig, _ := _GetStructSig(sig, 0)
		for i, s := range structSig {
			_AppendValue(buff, string(s), val.([]interface{})[i])
		}
		sigOffset = 2 + len(structSig)

	case '{':
		_AppendAlign(8, buff)
		dictSig, _ := _GetDictSig(sig, 0)
		for i, s := range dictSig {
			_AppendValue(buff, string(s), val.([]interface{})[i])
		}
		sigOffset = 2 + len(dictSig)
	}

	return
}

func _AppendParamsData(buff *bytes.Buffer, sig string, params *vector.Vector) {
	sigOffset := 0
	prmsOffset := 0
	for ; sigOffset < len(sig); prmsOffset++ {
		offset, _ := _AppendValue(buff, sig[sigOffset:len(sig)], params.At(prmsOffset))
		sigOffset += offset
	}
}

func _GetByte(buff []byte, index int) (byte, os.Error) {
	if len(buff) <= index {
		return 0, os.NewError("index error")
	}
	return buff[index], nil
}

func _GetInt16(buff []byte, index int) (int16, os.Error) {
	if len(buff) <= index+2-1 {
		return 0, os.NewError("index error")
	}
	var n int16
	e := binary.Read(bytes.NewBuffer(buff[index:len(buff)]), binary.LittleEndian, &n)
	if e != nil {
		return 0, e
	}
	return n, nil
}

func _GetUint16(buff []byte, index int) (uint16, os.Error) {
	if len(buff) <= index+2-1 {
		return 0, os.NewError("index error")
	}
	var q uint16
	e := binary.Read(bytes.NewBuffer(buff[index:len(buff)]), binary.LittleEndian, &q)
	if e != nil {
		return 0, e
	}
	return q, nil
}

func _GetInt32(buff []byte, index int) (int32, os.Error) {
	if len(buff) <= index+4-1 {
		return 0, os.NewError("index error")
	}
	var l int32
	e := binary.Read(bytes.NewBuffer(buff[index:len(buff)]), binary.LittleEndian, &l)
	if e != nil {
		return 0, e
	}
	return l, nil
}

func _GetUint32(buff []byte, index int) (uint32, os.Error) {
	if len(buff) <= index+4-1 {
		return 0, os.NewError("index error")
	}
	var u uint32
	e := binary.Read(bytes.NewBuffer(buff[index:len(buff)]), binary.LittleEndian, &u)
	if e != nil {
		return 0, e
	}
	return u, nil
}

func _GetBoolean(buff []byte, index int) (bool, os.Error) {
	if len(buff) <= index+4-1 {
		return false, os.NewError("index error")
	}
	var v int32
	e := binary.Read(bytes.NewBuffer(buff[index:len(buff)]), binary.LittleEndian, &v)
	if e != nil {
		return false, e
	}
	return 0 != v, nil
}

func _GetString(buff []byte, index int, size int) (string, os.Error) {
	if len(buff) <= (index + size - 1) {
		return "", os.NewError("index error")
	}
	return string(buff[index : index+size]), nil
}

func _GetStructSig(sig string, startIdx int) (string, os.Error) {
	if len(sig) <= startIdx || '(' != sig[startIdx] {
		return "<nil>", os.NewError("index error")
	}
	sigIdx := startIdx + 1
	for depth := 0; sigIdx < len(sig); sigIdx++ {
		switch sig[sigIdx] {
		case ')':
			if depth == 0 {
				return sig[startIdx+1 : sigIdx], nil
			}
			depth--
		case '(':
			depth++
		}
	}

	return "<nil>", os.NewError("parse error")
}

func _GetDictSig(sig string, startIdx int) (string, os.Error) {
	if len(sig) <= startIdx || '{' != sig[startIdx] {
		return "<nil>", os.NewError("index error")
	}
	sigIdx := startIdx + 1
	for depth := 0; sigIdx < len(sig); sigIdx++ {
		switch sig[sigIdx] {
		case '}':
			if depth == 0 {
				return sig[startIdx+1 : sigIdx], nil
			}
			depth--
		case '{':
			depth++
		}
	}

	return "<nil>", os.NewError("parse error")
}

func _GetSigBlock(sig string, index int) (string, os.Error) {
	switch sig[index] {
	case '(':
		str, e := _GetStructSig(sig, index)
		if e != nil {
			return "", e
		}
		return strings.Join([]string{"(", str, ")"}, ""), nil

	case '{':
		str, e := _GetDictSig(sig, index)
		if e != nil {
			return "", e
		}
		return strings.Join([]string{"{", str, "}"}, ""), nil

	}

	// default
	return sig[index : index+1], nil
}

func _GetVariant(buff []byte, index int) (valvec *vector.Vector, retidx int, e os.Error) {
	retidx = index
	sigSize := int(buff[retidx])
	retidx++
	sig := string(buff[retidx : retidx+sigSize])
	valvec, retidx, e = Parse(buff, sig, retidx+sigSize+1)
	return
}


func Parse(buff []byte, sig string, index int) (vec *vector.Vector, bufIdx int, err os.Error) {
	vec = new(vector.Vector)
	bufIdx = index
	for sigIdx := 0; sigIdx < len(sig); {
		switch sig[sigIdx] {
		case 'b': // bool
			bufIdx = _Align(4, bufIdx)
			b, e := _GetBoolean(buff, bufIdx)
			if e != nil {
				err = e
				return
			}
			vec.Push(b)
			bufIdx += 4
			sigIdx++

		case 'y': // byte
			v, e := _GetByte(buff, bufIdx)
			if e != nil {
				err = e
				return
			}
			vec.Push(v)
			bufIdx++
			sigIdx++

		case 'n': // int16
			bufIdx = _Align(2, bufIdx)
			n, e := _GetInt16(buff, bufIdx)
			if e != nil {
				err = e
				return
			}

			vec.Push(n)
			bufIdx += 2
			sigIdx++

		case 'q': // uint16
			bufIdx = _Align(2, bufIdx)
			q, e := _GetUint16(buff, bufIdx)
			if e != nil {
				err = e
				return
			}

			vec.Push(q)
			bufIdx += 2
			sigIdx++

		case 'u': // uint32
			bufIdx = _Align(4, bufIdx)

			u, e := _GetUint32(buff, bufIdx)
			if e != nil {
				err = e
				return
			}

			vec.Push(u)
			bufIdx += 4
			sigIdx++

		case 's', 'o': // string, object
			bufIdx = _Align(4, bufIdx)

			size, e := _GetInt32(buff, bufIdx)
			if e != nil {
				err = e
				return
			}

			str, e := _GetString(buff, bufIdx+4, int(size))
			if e != nil {
				err = e
				return
			}

			vec.Push(str)
			bufIdx += (4 + int(size) + 1)
			sigIdx++

		case 'g': // signature
			size, e := _GetByte(buff, bufIdx)
			if e != nil {
				err = e
				return
			}

			str, e := _GetString(buff, bufIdx+1, int(size))
			if e != nil {
				err = e
				return
			}
			vec.Push(str)
			bufIdx += (1 + int(size) + 1)
			sigIdx++

		case 'a': // array
			startIdx := _Align(4, bufIdx)
			arySize, e := _GetInt32(buff, startIdx)
			if e != nil {
				err = e
				return
			}

			sigBlock, e := _GetSigBlock(sig, sigIdx+1)
			if e != nil {
				err = e
				return
			}

			aryIdx := startIdx + 4
			aryVec := new(vector.Vector)
			for aryIdx < (startIdx+4)+int(arySize) {
				retvec, retidx, e := Parse(buff, sigBlock, aryIdx)
				if e != nil {
					err = e
					return
				}

				aryVec.AppendVector(retvec)
				aryIdx = retidx
			}
			bufIdx = aryIdx
			sigIdx += (1 + len(sigBlock))
			vec.Push(aryVec)

		case '(': // struct
			idx := _Align(8, bufIdx)
			stSig, e := _GetStructSig(sig, sigIdx)
			if e != nil {
				err = e
				return
			}

			retvec, retidx, e := Parse(buff, stSig, idx)
			if e != nil {
				err = e
				return
			}

			bufIdx = retidx
			sigIdx += (len(stSig) + 2)
			vec.Push(retvec)

		case '{': // dict
			idx := _Align(8, bufIdx)
			stSig, e := _GetDictSig(sig, sigIdx)
			if e != nil {
				err = e
				return
			}

			retvec, retidx, e := Parse(buff, stSig, idx)
			if e != nil {
				err = e
				return
			}

			bufIdx = retidx
			sigIdx += (len(stSig) + 2)
			vec.Push(retvec)

		case 'v': // variant
			val, idx, e := _GetVariant(buff, bufIdx)
			if e != nil {
				err = e
				return
			}

			bufIdx = idx
			sigIdx++
			vec.AppendVector(val)

		default:
			fmt.Println(sig[sigIdx])
			return nil, index, os.NewError("unknown type")
		}
	}
	return
}
