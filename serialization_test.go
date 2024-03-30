package ui

import (
	"reflect"
	"testing"
)

type Person struct {
	Name    string
	Age     int
	Hobbies []string
}

func TestSerializationAndDeserialization(t *testing.T) {
	tests := []struct {
		name    string
		person  Person
		wantErr bool
	}{
		{
			name: "Simple Person",
			person: Person{
				Name:    "John Doe",
				Age:     30,
				Hobbies: []string{"reading", "cycling"},
			},
			wantErr: false,
		},
		// Add more test cases for different scenarios, including edge cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			serializedValue, err := Serialize(tt.person)
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Deserialize
			var deserializedPerson Person
			if err := Deserialize(serializedValue, &deserializedPerson); err != nil {
				if !tt.wantErr {
					t.Errorf("Deserialize() error = %v", err)
				}
				return
			}

			// Compare the original and deserialized persons
			if !reflect.DeepEqual(tt.person, deserializedPerson) {
				t.Errorf("Original and deserialized persons are not equal. Original: %+v, Deserialized: %+v", tt.person, deserializedPerson)
			}
		})
	}
}
