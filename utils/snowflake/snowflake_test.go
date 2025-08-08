package snowflake

import "testing"

func TestSetupSnowflake(t *testing.T) {
	err := Setup(0)
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateSnowflake(t *testing.T) {
	_, err := Generate()
	if err != nil {
		t.Error(err)
	}
}

func TestSnowflakeIncrementOverflow(t *testing.T) {
	for range 100000 {
		_, err := Generate()
		if err != nil {
			return
		}
	}
	t.Error("Expected increment overflow, but there wasn't")
}
