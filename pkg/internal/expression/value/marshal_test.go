package value

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalJSON(t *testing.T) {
	notOne, _ := Number(1).Negate()
	obj := MustMap(
		String("i got"), Number(99),
		String("problems"), String("but an"),
		String("array"), NewList(notOne, NewList(), MustMap(), Null(), Func(nil), Bool(true), Bool(false)),
		String("simple"), NewList(Number(1), Number(2)),
	)

	data, err := json.Marshal(obj)
	require.NoError(t, err)
	assert.Equal(t, `{"i got":99,"problems":"but an","array":[-1,[],{},null,"\u003cfunc\u003e",true,false],"simple":[1,2]}`, string(data))
}
