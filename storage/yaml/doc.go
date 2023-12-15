// package yaml provides a read only storage implementation which sources its URLs from a YAML file. THe file is
// expected to be in the format:
//
//	---
//	- from: //s3k/foo
//	- to: //s3k/bar
//
// Where there is no scheme (assumed to be the default case), the "//" is required to clearly indicate this is a
// schemeless URL.
package yaml
