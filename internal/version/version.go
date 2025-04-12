package version

import "strconv"

type Version struct {
	Major      int
	Minor      int
	FileNumber int
}

func CompareVersion(src, dst Version) int {
	if src.Major < dst.Major {
		return -1
	} else if src.Major > dst.Major {
		return 1
	}

	if src.Minor < dst.Minor {
		return -1
	} else if src.Minor > dst.Minor {
		return 1
	}

	return 0
}

func CompareVersionWithFile(src, dst Version) int {
	if res := CompareVersion(src, dst); res != 0 {
		return res
	}

	if src.FileNumber < dst.FileNumber {
		return -1
	} else if src.FileNumber > dst.FileNumber {
		return 1
	}

	return 0
}

func ParseStringToVersion(major, minor, fileNumber string) (Version, error) {
	var v Version

	majorInt, err := strconv.Atoi(major)
	if err != nil {
		return v, err
	}
	minorInt, err := strconv.Atoi(minor)
	if err != nil {
		return v, err
	}
	fileNumberInt, err := strconv.Atoi(fileNumber)
	if err != nil {
		return v, err
	}

	v.Major = majorInt
	v.Minor = minorInt
	v.FileNumber = fileNumberInt

	return v, nil
}
