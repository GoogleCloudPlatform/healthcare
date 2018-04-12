# DICOM test files

This directory contains generated DICOM data used in unit tests. The .dcm files
are generated from the .txt and .jpg files found in the directory. This is done
by using the DCMTK [dump2dcm](http://support.dcmtk.org/docs/dump2dcm.html) tool.

## Example:

```shell
dump2dcm mr_1.0.0.0.txt mr_1.0.0.0.dcm
img2dcm image.jpg img_1.0.0.0.dcm
```
