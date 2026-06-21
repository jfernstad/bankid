package bankid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLanguageCases(t *testing.T) {

	validCases := []string{
		"en", "EN", "eN",
		"se", "SE", "sE",
		"sv", "SV", "Sv", // v6.0: also accept "sv" for Swedish
	}

	for _, c := range validCases {
		_, err := NewMessages(c)
		assert.Nil(t, err)
	}
}

func TestUnknownLanguages(t *testing.T) {
	_, err := NewMessages("es") // No spanish support yet
	assert.NotNil(t, err)
}

func TestCorrectLanguages(t *testing.T) {
	// English
	en, err := NewMessages("en")
	assert.Nil(t, err)
	assert.Equal(t, messages_EN[RFA1], en.Msg(RFA1))

	// Swedish (se)
	se, err := NewMessages("se")
	assert.Nil(t, err)
	assert.Equal(t, messages_SE[RFA1], se.Msg(RFA1))

	// Swedish (sv)
	sv, err := NewMessages("sv")
	assert.Nil(t, err)
	assert.Equal(t, messages_SE[RFA1], sv.Msg(RFA1))
}

func TestInvalidMessagesReference(t *testing.T) {
	se, err := NewMessages("se")
	assert.Nil(t, err)
	assert.Equal(t, "", se.Msg("INVALID_REFERENCE"))
}

func TestRFA23Exists(t *testing.T) {
	en, err := NewMessages("en")
	assert.Nil(t, err)
	assert.NotEmpty(t, en.Msg(RFA23))

	se, err := NewMessages("se")
	assert.Nil(t, err)
	assert.NotEmpty(t, se.Msg(RFA23))
}

func TestAllMessagesPresent(t *testing.T) {
	allCodes := []string{
		RFA1, RFA2, RFA3, RFA4, RFA5, RFA6, RFA8, RFA9,
		RFA13, RFA14_A, RFA14_B, RFA15_A, RFA15_B, RFA16,
		RFA17_A, RFA17_B, RFA18, RFA19, RFA20,
		RFA21, RFA22, RFA23,
	}

	en, _ := NewMessages("en")
	se, _ := NewMessages("se")

	for _, code := range allCodes {
		assert.NotEmpty(t, en.Msg(code), "English message missing for %s", code)
		assert.NotEmpty(t, se.Msg(code), "Swedish message missing for %s", code)
	}
}
