package index

import (
	"github.com/balzaczyy/golucene/core/index/model"
	"github.com/balzaczyy/golucene/core/util"
)

// index/TermsHashConsumer.java

type TermsHashConsumer interface {
	flush(map[string]TermsHashConsumerPerField, SegmentWriteState) error
	abort()
	startDocument()
	addField(*TermsHashPerField, model.FieldInfo) TermsHashConsumerPerField
}

// index/TermVectorsConsumer.java

type TermVectorsConsumer struct {
	writer    TermVectorsWriter
	docWriter *DocumentsWriterPerThread
	docState  *docState

	hasVectors       bool
	numVectorsFields int
	lastDocId        int
	perFields        []*TermVectorsConsumerPerField

	lastVectorFieldName string
}

func newTermVectorsConsumer(docWriter *DocumentsWriterPerThread) *TermVectorsConsumer {
	return &TermVectorsConsumer{
		docWriter: docWriter,
		docState:  docWriter.docState,
	}
}

func (tvc *TermVectorsConsumer) flush(fieldsToFlush map[string]TermsHashConsumerPerField,
	state SegmentWriteState) (err error) {
	if tvc.writer != nil {
		numDocs := state.segmentInfo.DocCount()
		assert(numDocs > 0)
		// At least one doc in this run had term vectors enabled
		func() {
			defer func() {
				err = mergeError(err, util.Close(tvc.writer))
				tvc.writer = nil
				tvc.lastDocId = 0
				tvc.hasVectors = false
			}()

			err = tvc.fill(numDocs)
			if err == nil {
				err = tvc.writer.finish(state.fieldInfos, numDocs)
			}
		}()
		if err != nil {
			return err
		}
	}

	for _, field := range fieldsToFlush {
		perField := field.(*TermVectorsConsumerPerField)
		perField.termsHashPerField.reset()
		perField.shrinkHash()
	}
	return
}

/*
Fills in no-term-vectors for all docs we haven't seen since the last
doc that had term vectors.
*/
func (c *TermVectorsConsumer) fill(docId int) error {
	for c.lastDocId < docId {
		c.writer.startDocument(0)
		err := c.writer.finishDocument()
		if err != nil {
			return err
		}
		c.lastDocId++
	}
	return nil
}

func (tvc *TermVectorsConsumer) abort() {
	tvc.hasVectors = false

	if tvc.writer != nil {
		tvc.writer.abort()
		tvc.writer = nil
	}

	tvc.lastDocId = 0
	tvc.reset()
}

func (tvc *TermVectorsConsumer) reset() {
	tvc.perFields = nil
	tvc.numVectorsFields = 0
}

func (tvc *TermVectorsConsumer) addField(termsHashPerField *TermsHashPerField,
	fieldInfo model.FieldInfo) TermsHashConsumerPerField {
	panic("not implemented yet")
}

func (c *TermVectorsConsumer) startDocument() {
	assert(c.clearLastVectorFieldName())
	c.reset()
}

func (c *TermVectorsConsumer) clearLastVectorFieldName() bool {
	c.lastVectorFieldName = ""
	return true
}

// index/FreqProxTermsWriter.java

type FreqProxTermsWriter struct {
}

func (w *FreqProxTermsWriter) abort() {}

// TODO: would be nice to factor out more of this, e.g. the
// FreqProxFieldMergeState, and code to visit all Fields under the
// same FieldInfo together, up into TermsHash*. Other writers would
// presumably share a lot of this...

func (w *FreqProxTermsWriter) flush(fieldsToFlush map[string]TermsHashConsumerPerField,
	state SegmentWriteState) (err error) {
	// Gather all FieldData's that have postings, across all ThreadStates
	var allFields []*FreqProxTermsWriterPerField

	for _, f := range fieldsToFlush {
		if perField := f.(*FreqProxTermsWriterPerField); perField.termsHashPerField.bytesHash.Size() > 0 {
			allFields = append(allFields, perField)
		}
	}

	// Sort by field name
	util.IntroSort(FreqProxTermsWriterPerFields(allFields))

	var consumer FieldsConsumer
	consumer, err = state.segmentInfo.Codec().(Codec).PostingsFormat().FieldsConsumer(state)
	if err != nil {
		return err
	}

	var success = false
	defer func() {
		if success {
			err = mergeError(err, util.Close(consumer))
		} else {
			util.CloseWhileSuppressingError(consumer)
		}
	}()

	var termsHash *TermsHash
	// Current writer chain:
	// FieldsConsumer
	// -> IMPL: FormatPostingsTermsDictWriter
	// -> TermsConsumer
	// -> IMPL: FormatPostingsTermsDictWriter.TermsWriter
	// -> DocsConsumer
	// -> IMPL: FormatPostingsDocWriter
	// -> PositionsConsumer
	// -> IMPL: FormatPostingsPositionsWriter

	for _, fieldWriter := range allFields {
		fieldInfo := fieldWriter.fieldInfo

		// If this field has postings then add them to the segment
		err = fieldWriter.flush(fieldInfo.Name, consumer, state)
		if err != nil {
			return err
		}

		perField := fieldWriter.termsHashPerField
		assert(termsHash == nil || termsHash == perField.termsHash)
		termsHash = perField.termsHash
		numPostings := perField.bytesHash.Size()
		perField.reset()
		perField.shrinkHash(numPostings)
		fieldWriter.reset()
	}

	if termsHash != nil {
		termsHash.reset()
	}
	success = true
	return
}

type FreqProxTermsWriterPerFields []*FreqProxTermsWriterPerField

func (a FreqProxTermsWriterPerFields) Len() int      { return len(a) }
func (a FreqProxTermsWriterPerFields) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a FreqProxTermsWriterPerFields) Less(i, j int) bool {
	return a[i].fieldInfo.Name < a[j].fieldInfo.Name
}

func (w *FreqProxTermsWriter) addField(termsHashPerField *TermsHashPerField,
	fieldInfo model.FieldInfo) TermsHashConsumerPerField {
	panic("not implemented yet")
}

func (w *FreqProxTermsWriter) startDocument() {}
