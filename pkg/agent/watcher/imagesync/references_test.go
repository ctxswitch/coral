package imagesync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReferences(t *testing.T) {
	c := NewReferences()
	c.Add("uuid1", "name1", "digest1")
	assert.Equal(t, 1, c.images["name1"].References("digest1"))

	// Add the same image again.  We still should only have a single reference.
	c.Add("uuid1", "name1", "digest1")
	assert.Equal(t, 1, c.images["name1"].References("digest1"))

	// Add a different image.  We should have two references.
	c.Add("uuid2", "name1", "digest1")
	assert.Equal(t, 2, c.images["name1"].References("digest1"))

	// Removing the reference to the first image.  We should still have a single
	// reference to the second image.
	c.Remove("uuid1", "name1", "digest1")
	assert.Equal(t, 1, c.images["name1"].References("digest1"))

	// Removing the reference to the second image.  We should have no references.
	c.Remove("uuid2", "name1", "digest1")
	assert.Equal(t, 0, c.images["name1"].References("digest1"))

	// Removing the reference to the second image again should still be zero.
	c.Remove("uuid2", "name1", "digest1")
	assert.Equal(t, 0, c.images["name1"].References("digest1"))

	// Both references are back.
	c.Add("uuid1", "name1", "digest1")
	c.Add("uuid2", "name1", "digest1")
	assert.Equal(t, 2, c.images["name1"].References("digest1"))

	// One of the resources is updated.  We should now have one reference to the
	// new image and one reference to the old image.
	c.Add("uuid1", "name1", "digest2")
	assert.Equal(t, 1, c.images["name1"].References("digest2"))
	assert.Equal(t, 1, c.images["name1"].References("digest1"))

	// Removing the first image should still leave us with a reference to the second image.
	c.Remove("uuid1", "name1", "digest2")
	assert.Equal(t, 0, c.images["name1"].References("digest2"))
	assert.Equal(t, 1, c.images["name1"].References("digest1"))
}
