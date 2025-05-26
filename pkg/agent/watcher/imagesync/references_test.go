package imagesync

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type ReferencesTestSuite struct {
	suite.Suite
}

func (s *ReferencesTestSuite) SetupTest() {}

func (s *ReferencesTestSuite) TearDownTest() {}

func TestReferencesTestSuite(t *testing.T) {
	suite.Run(t, new(ReferencesTestSuite))
}

func (s *ReferencesTestSuite) TestReferences() {
	c := NewReferences()
	c.Add("uid1", "name1", "digest1")
	s.Equal(1, c.images["name1"].References("digest1"))

	// Add the same image again.  We still should only have a single reference.
	c.Add("uid1", "name1", "digest1")
	s.Equal(1, c.images["name1"].References("digest1"))

	// Add a different image.  We should have two references.
	c.Add("uid2", "name1", "digest1")
	s.Equal(2, c.images["name1"].References("digest1"))

	// Removing the reference to the first image.  We should still have a single
	// reference to the second image.
	c.Remove("uid1", "name1", "digest1")
	s.Equal(1, c.images["name1"].References("digest1"))

	// Removing the reference to the second image.  We should have no references.
	c.Remove("uid2", "name1", "digest1")
	s.Equal(0, c.images["name1"].References("digest1"))

	// Removing the reference to the second image again should still be zero.
	c.Remove("uid2", "name1", "digest1")
	s.Equal(0, c.images["name1"].References("digest1"))

	// Both references are back.
	c.Add("uid1", "name1", "digest1")
	c.Add("uid2", "name1", "digest1")
	s.Equal(2, c.images["name1"].References("digest1"))

	// One of the resources is updated.  We should now have one reference to the
	// new image and one reference to the old image.
	c.Add("uid1", "name1", "digest2")
	s.Equal(1, c.images["name1"].References("digest2"))
	s.Equal(1, c.images["name1"].References("digest1"))

	// Removing the first image should still leave us with a reference to the second image.
	c.Remove("uid1", "name1", "digest2")
	s.Equal(0, c.images["name1"].References("digest2"))
	s.Equal(1, c.images["name1"].References("digest1"))
}

func (s *ReferencesTestSuite) TestReferences_ToImageList() {
	// Test the ToImageList method to ensure it returns unique images.
	c := NewReferences()
	c.Add("uid1", "name1", "digest1")
	c.Add("uid2", "name2", "digest2")
	c.Add("uid3", "name1", "digest1")
	c.Add("uid4", "name3", "digest3")
	c.Add("uid5", "name2", "digest2")

	images := c.ToImageList()
	s.ElementsMatch([]string{"name1", "name2", "name3"}, images)

	// Remove all references to name2 and check the list again
	c.Remove("uid2", "name2", "digest2")
	c.Remove("uid5", "name2", "digest2")
	images = c.ToImageList()
	s.ElementsMatch([]string{"name1", "name3"}, images)

	// Remove all references to name1 and name3
	c.Remove("uid1", "name1", "digest1")
	c.Remove("uid3", "name1", "digest1")
	c.Remove("uid4", "name3", "digest3")
	images = c.ToImageList()
	s.Empty(images)
}

func (s *ReferencesTestSuite) TestReferences_ImageListForUID() {
	c := NewReferences()
	c.Add("uid1", "name1", "digest1")
	c.Add("uid1", "name2", "digest2")
	c.Add("uid2", "name1", "digest1")
	c.Add("uid2", "name2", "digest2")
	c.Add("uid2", "name3", "digest3")

	expected := []string{"name1", "name2"}
	images := c.ImageListForUID("uid1")
	s.ElementsMatch(expected, images)

	expected = []string{"name1", "name2", "name3"}
	images = c.ImageListForUID("uid2")
	s.ElementsMatch(expected, images)

	expected = []string{}
	images = c.ImageListForUID("uid3")
	s.ElementsMatch(expected, images)
}

func (s *ReferencesTestSuite) TestReferences_HasUID() {
	c := NewReferences()
	c.Add("uid1", "name1", "digest1")
	c.Add("uid2", "name2", "digest2")

	// Check if the references have the correct UID
	s.True(c.HasUID("uid1"))
	s.True(c.HasUID("uid2"))
	s.False(c.HasUID("uid3"))

	// Remove a reference and check again
	c.Remove("uid1", "name1", "digest1")
	s.False(c.HasUID("uid1"))
	s.True(c.HasUID("uid2"))
}
