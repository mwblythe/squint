package squint

func (s *SquintSuite) TestCompat() {
	b1 := NewBuilder(EmptyValues(false), NilValues(false))
	b2 := NewBuilder(EmptyValues(true), NilValues(true))

	// test maps
	var bar *bool
	s.True(b1.HasValues(H{"age": 10}))
	s.True(b1.HasValues(H{"age": 0}))
	s.False(b1.HasValues(H{"name": ""}))
	s.False(b1.HasValues(H{"foo": nil}))
	s.False(b1.HasValues(H{"foo": bar}))
	s.False(b1.HasValues(H{}))

	s.True(b1.HasValues(H{"age": 0}))
	s.True(b2.HasValues(H{"name": ""}))
	s.True(b2.HasValues(H{"foo": nil}))
	s.True(b2.HasValues(H{"foo": bar}))

	// test structs
	var junk struct{}
	type trunk struct {
		Name string
	}

	s.False(b1.HasValues(junk))
	s.False(b1.HasValues(trunk{}))
	s.True(b1.HasValues(trunk{"Frank"}))
	s.True(b2.HasValues(trunk{}))

	// test other
	s.True(b1.HasValues("hello"))
}
