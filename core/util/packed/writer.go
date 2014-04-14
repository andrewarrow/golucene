package packed

import (
	"errors"
)

// util/packed/PackedInts.java#Writer

/* A write-once Writer. */
type Writer interface {
	// Add a value to the stream.
	Add(v int64) error
	// The number of bits per value.
	BitsPerValue() int
	// Perform end-of-stream operations.
	Finish() error
}

type WriterImpl struct {
	out          DataOutput
	valueCount   int
	bitsPerValue int
}

func newWriter(out DataOutput, valueCount, bitsPerValue int) *WriterImpl {
	assert(bitsPerValue <= 64)
	assert(valueCount >= 0 || valueCount == -1)
	return &WriterImpl{out, valueCount, bitsPerValue}
}

func (w *WriterImpl) BitsPerValue() int {
	return w.bitsPerValue
}

// util/packed/PackedWriter.java

type PackedWriter struct {
	*WriterImpl
	finished   bool
	format     PackedFormat
	encoder    BulkOperation
	nextBlocks []byte
	nextValues []int64
	iterations int
	off        int
	written    int
}

func newPackedWriter(format PackedFormat, out DataOutput,
	valueCount, bitsPerValue, mem int) *PackedWriter {
	encoder := newBulkOperation(format, uint32(bitsPerValue))
	iterations := encoder.computeIterations(valueCount, mem)
	return &PackedWriter{
		WriterImpl: newWriter(out, valueCount, bitsPerValue),
		format:     format,
		encoder:    encoder,
		iterations: iterations,
		nextBlocks: make([]byte, iterations*encoder.ByteBlockCount()),
		nextValues: make([]int64, iterations*encoder.ByteValueCount()),
	}
}

func (w *PackedWriter) Add(v int64) error {
	assert2(w.bitsPerValue == 64 || (v >= 0 && v <= MaxValue(w.bitsPerValue)),
		"%v", w.bitsPerValue)
	assert(!w.finished)
	if w.valueCount != -1 && w.written >= w.valueCount {
		return errors.New("Writing past end of stream")
	}
	w.nextValues[w.off] = v
	if w.off++; w.off == len(w.nextValues) {
		err := w.flush()
		if err != nil {
			return err
		}
	}
	w.written++
	return nil
}

func (w *PackedWriter) Finish() error {
	panic("not implemented yet")
}

func (w *PackedWriter) flush() error {
	w.encoder.encodeLongToByte(w.nextValues, w.nextBlocks, w.iterations)
	blockCount := int(w.format.ByteCount(PACKED_VERSION_CURRENT,
		int32(w.off), uint32(w.bitsPerValue)))
	err := w.out.WriteBytes(w.nextBlocks[:blockCount])
	if err != nil {
		return err
	}
	for i, _ := range w.nextValues {
		w.nextValues[i] = 0
	}
	w.off = 0
	return nil
}
