package encoding

import "testing"

func TestMarshalUnmarshal(t *testing.T) {
	tests := []string{"", "The Quick Brown Fox Jumps Over The Lazy Dog", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Ut gravida mattis urna a tincidunt. Sed urna lacus, pretium a dolor sit amet, eleifend ornare urna. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Nulla aliquet pulvinar sem eu tincidunt. Nulla lobortis pretium urna, at fringilla lectus aliquam ut. Mauris eu ante ac ligula consequat ullamcorper. Aenean eros urna, commodo ut iaculis id, malesuada lacinia velit. Integer blandit nibh vitae diam accumsan, sed efficitur dolor egestas. Nullam tempus venenatis neque."}
	asn := uint32(34553)

	for _, testString := range tests {
		t.Run("marshal/unmarshal: "+testString, func(t *testing.T) {
			marshalled := Marshal(testString, asn)
			unmarshalled := Unmarshal(marshalled, asn)
			if unmarshalled != testString {
				t.Errorf("marshal/unmarshal got \"%s\" expected \"%s\"", unmarshalled, testString)
			}
		})
	}
}
